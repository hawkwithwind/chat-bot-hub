# init docker enviornment

```bash

# run the following command to init the docker-compose file

quickStart/setup.sh 

# then up all services

make && docker-compose up -d

# run the mysql migrate scripts

TODO:


```


# init chathub user as chathub/chathub (username/password)
docker-compose exec mysql mysql -u chathub -pchathub --database chathub -e "insert into accounts(accountid,accountname,secret,createat,updateat)values('544ca666430c0a42a386c7da4afba1f7','chathub','5c4d4b34035751c77a8bc5d1665b8d9e8229381da492aaed5a5855b49f8d6df3',now(),now())"


# other quick command 

## query chathub mysql db
docker-compose exec mysql mysql -u chathub -pchathub --database chathub

## in case you want to use an external mysql database, you can run migrate as follows

docker run -v ${PWD}/migrate:/migrations --network host migrate/migrate \
    -path=/migrations/ -database mysql://USERNAME:PASSWD@(DBHOST:DBPORT)/DATABASENAME up
