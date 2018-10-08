package main

/*
status [000] -> GRPC connected, but not logged in
status [100] -> have qq number, wait for login response
status [150] -> wait for scan qr code
status [200] -> logged in
status [500] -> GPRC connection closed, wait for client start again
*/

type ChatBot struct {
	ClientId   string
	ClientType string
	Name       string
	StartAt    int64
	LastPing   int64
	Login      int64
	Status     int32
}
