## sorcia
Sorcia is a self-hosted and minimal web frontend for git repositories written in Go. This project is under development.

### pre-requisites
  - Ubuntu 18.04 LTS
  - PostgreSQL 11
  - Go 1.13
 
### installation
Let's create a user and databse on PostgreSQL 11
```
CREATE DATABASE sorciadb;
CREATE USER sorcia WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE sorciadb to sorcia;
```

Now create a `git` user on your machine and clone the sorcia repository
```
sudo adduser --disabled-login --gecos 'sorcia' git
git clone --depth 1 https://github.com/getsorcia/sorcia.git sorcia
sudo su - git
```

Move to sorcia directory and change the `config/app.ini` file to match with the database, user and password that we had created above.
```
cd sorcia
```

Open `config/app.ini` with your favorite editor
```
[postgres]
hostname = localhost # localhost or external IP of your postgresql database server
port = 5432
name = sorciadb
username = sorcia
password = your-secure-password
sslmode = disable # either "disable", "require" or "verify-full"
```

Now that we have configured `app.ini`, let's build and start the sorcia web server from the project root which is `sorcia`.
```
go build sorcia.go
./sorcia web
```
That's it, sorcia will run on port `1937` (you can configure this in `app.ini`). So, if you are running locally - it should be `http://localhost:1937`
