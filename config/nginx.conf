server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name example.com www.example.com;
    return 301 https://example.com$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name www.example.com;

    ssl on;
    ssl_certificate /etc/ssl/example.com.pem;
    ssl_certificate_key /etc/ssl/example.com.key;

    return 301 https://example.com$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name example.com;

    ssl on;
    ssl_certificate /etc/ssl/example.com.pem;
    ssl_certificate_key /etc/ssl/example.com.key;

    location / {
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header Host $host;
        proxy_pass http://127.0.0.1:1937;
    }
}