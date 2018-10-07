package main

import (
	"fmt"
	"log"	
	//"io"
	"os"
	"time"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
    "github.com/gorilla/context"
    "github.com/mitchellh/mapstructure"
	"github.com/garyburd/redigo/redis"

	"google.golang.org/grpc"
	grpcctx "golang.org/x/net/context"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	//"google.golang.org/grpc/reflection"

)


type RedisConfig struct {
	Host string
	Port string
	Db string
}

type WebConfig struct {
	Host string
	Port string
	User string
	Pass string
	SecretPhrase string
	Redis RedisConfig
}

type CommonResponse struct {
	Code int `json:"code"`
	Message string `json:"message,omitempty"`
	Ts int64 `json:"ts"`
	Error ErrorMessage `json:"error,omitempty""`
	Body interface{} `json:"body,omitempty""`
}

type ErrorMessage struct {
    Message string `json:"message,omitempty"` 
}

type User struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type JwtToken struct {
    Token string `json:"token"`
}

type WebServer struct {
	logger *log.Logger
	redispool *redis.Pool
	config WebConfig
	hubport string
}

func (ctx *WebServer) init() {
	ctx.logger = log.New(os.Stdout, "[WEB] ", log.Ldate | log.Ltime | log.Lshortfile)
	ctx.redispool = ctx.newRedisPool(
		fmt.Sprintf("%s:%s", ctx.config.Redis.Host, ctx.config.Redis.Port),
		ctx.config.Redis.Db)
}

func (ctx *WebServer) Info(msg string) {
	ctx.logger.Printf(msg)
}

func (ctx *WebServer) Infof(msg string, v ... interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *WebServer) Error(msg string) {
	ctx.logger.Fatalf(msg)
}

func (ctx *WebServer) Errorf(msg string, v ... interface{}) {
	ctx.logger.Fatalf(msg, v...)
}

func (ctx *WebServer) deny(w http.ResponseWriter, msg string) {
	// HTTP CODE 403
	w.WriteHeader(http.StatusForbidden)	
	json.NewEncoder(w).Encode(CommonResponse{
		Code: -1,
		Message: msg,
		Ts: time.Now().Unix(),
	})
}

func (ctx *WebServer) complain(w http.ResponseWriter, code int, msg string) {
	// HTTP CODE 400
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(CommonResponse{
		Code: code,
		Ts: time.Now().Unix(),
		Error: ErrorMessage{Message: msg},
	})
}

func (ctx *WebServer) ok(w http.ResponseWriter, msg string, body interface{}) {
	json.NewEncoder(w).Encode(CommonResponse{
		Code: 0,
		Ts: time.Now().Unix(),
		Message: msg,
		Body: body,
	})
}


func (ctx *WebServer) fail(w http.ResponseWriter, msg string) {
	// HTTP CODE 500
	w.WriteHeader(http.StatusInternalServerError);
	json.NewEncoder(w).Encode(CommonResponse{
		Code: -1,
		Ts: time.Now().Unix(),
		Error: ErrorMessage{Message: msg},
	})
}

func (ctx *WebServer) hello(w http.ResponseWriter, r *http.Request) {
	ctx.ok(w, "hello", nil)
}

func (ctx *WebServer) getBots(w http.ResponseWriter, r *http.Request) {
	ctx.Infof("Dial %s", fmt.Sprintf("127.0.0.1:%s", ctx.hubport))
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%s", ctx.hubport), grpc.WithInsecure())
	if err != nil {
		ctx.fail(w, fmt.Sprintf("无法连接本地RPC %s", err.Error()))
		return
	}
	defer conn.Close();
	client := pb.NewChatBotHubClient(conn)

	gctx, cancel := grpcctx.WithTimeout(grpcctx.Background(), 10*time.Second)
	defer cancel()	
	botsreply, err := client.GetBots(gctx, &pb.BotsRequest{Secret: "secret"})
	if err != nil {
		ctx.fail(w, err.Error())
		return
	}
	
	ctx.ok(w, "", botsreply)
}

func (ctx *WebServer) login(w http.ResponseWriter, req *http.Request) {
	var user User
    _ = json.NewDecoder(req.Body).Decode(&user)

	if user.Username != ctx.config.User || user.Password != ctx.config.Pass {
		ctx.deny(w, "用户名密码不匹配")
		return
	}
	
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "username": user.Username,
        "password": user.Password,
    })
    tokenString, error := token.SignedString([]byte(ctx.config.SecretPhrase))
    if error != nil {
		ctx.fail(w, error.Error())
		return
    }

    ctx.ok(w, "登录成功", JwtToken{Token: tokenString})
}

func (ctx *WebServer) validate(next http.HandlerFunc) http.HandlerFunc {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        authorizationHeader := req.Header.Get("authorization")
        if authorizationHeader != "" {
            bearerToken := strings.Split(authorizationHeader, " ")
            if len(bearerToken) == 2 {
                token, error := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
                    if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                        return nil, fmt.Errorf("There was an error")
                    }
                    return []byte(ctx.config.SecretPhrase), nil
                })
                if error != nil {
					ctx.fail(w, error.Error())
                    return
                }
                if token.Valid {
					var user User
					mapstructure.Decode(token.Claims, &user)
					if user.Username != ctx.config.User || user.Password != ctx.config.Pass {
						ctx.deny(w, "用户名密码不匹配")
						return
					}
					
                    context.Set(req, "decoded", token.Claims)
                    next(w, req)
                } else {
					ctx.deny(w, "身份令牌无效")
                }
            }
        } else {
			ctx.deny(w, "未登录用户无权限访问")
		}
    })
}

func (ctx *WebServer) newRedisPool(server string, db string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:3,
		IdleTimeout: 240 *time.Second,
		Dial: func() (redis.Conn, error) {
			c, err:= redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("SELECT", db); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}


func (ctx *WebServer) serve() {
	ctx.init()
	
	r := mux.NewRouter()
	r.HandleFunc("/hello", ctx.validate(ctx.hello)).Methods("GET")
	r.HandleFunc("/bots", ctx.getBots).Methods("GET")
	r.HandleFunc("/login", ctx.login).Methods("POST")	
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("/app/static/")))
	
	ctx.Info("restful server starts.")
	addr := fmt.Sprintf("%s:%s" , ctx.config.Host, ctx.config.Port)
	ctx.Infof("listen %s.", addr)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		ctx.Errorf("failed %v", err)
	}

	ctx.Info("Server stopped")
}
