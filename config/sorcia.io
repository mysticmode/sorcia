server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name sorcia.io www.sorcia.io;
    return 301 https://sorcia.io$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name www.sorcia.io;

    ssl on;
    ssl_certificate /etc/ssl/sorcia.io.pem;
    ssl_certificate_key /etc/ssl/sorcia.io.key;

    return 301 https://sorcia.io$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name sorcia.io;

    ssl on;
    ssl_certificate /etc/ssl/sorcia.io.pem;
    ssl_certificate_key /etc/ssl/sorcia.io.key;

    location / {
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header Host $host;
        proxy_pass http://127.0.0.1:1937;
    }
}