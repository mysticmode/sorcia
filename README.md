## sorcia
Sorcia is a self-hosted web frontend for git repositories which is written in Golang.

### community
 * Ask your questions on [sorcia@googlegroups.com](https://groups.google.com/d/forum/sorcia).
 * Subscribe to release announcements on [sorcia-announce@googlegroups.com](https://groups.google.com/d/forum/sorcia-announce).
 * Send patches to [sorcia-devel@googlegroups.com](https://groups.google.com/d/forum/sorcia-devel).

### pre-requisites
 * Ubuntu 18.04 LTS
 * SQLite3
 
### installation
At this moment, this documentation assumes that you are using a freshly installed Ubuntu 18.04 LTS box for the Sorcia installation. This will get updated to the platforms that Go can compile for as soon as possible. If you had installed Sorcia on a different OS or Architecture successfully, please feel free to send patches and update this README section.

SSH into your server as a root user or a user who has root privleges. For commands that needs root privileges, this documentation will prefix the command with `sudo`.

First, update and upgrade your OS utilities and packages.
```
sudo apt update
sudo apt upgrade
```

Install the necessary packages in order to run the Sorcia binary on your server.
```
sudo apt -y install software-properties-common build-essential git-core sqlite3 wget vim nginx
```

Now create a `git` user on your machine.
```
sudo adduser --disabled-login --gecos 'sorcia' git
sudo su - git
```

**Install from binary**
```
wget https://sorcia.mysticmode.org/dl/sorcia-0.2.1-linux-amd64.tar.gz
mkdir sorcia
tar -C sorcia -xzf sorcia.linux-amd64.tar.gz
cd sorcia
chmod +x sorcia
```

**(or) Install from source**
Download Go 1.14 from [https://golang.org/dl/](https://golang.org/dl/) using `wget`.
```
mkdir go local
tar -C local -xzf <go.tar.gz>
```

Open `.bashrc` using your favorite editor and add these lines at the bottom of the file.
```
export PATH=$PATH:$HOME/local/go/bin
export GOPATH=$HOME/go
source ~/.bashrc
```

Now download the Sorcia repository and build from source.
```
git clone https://git.mysticmode.org/r/sorcia.git
cd sorcia
go build sorcia.go
chmod +x sorcia
```

When you are in the project root directory,
```
cp config/app.ini.sample config/app.ini
```
and change the `app.ini` config file if you only prefer. Otherwise the default config is fine to go with.

Move back to the root user or user with root privileges with `exit` command. Change the SSH port from `22` to something else. Sorcia by default config which is in `config/app.ini` will run the Git SSH server on port 22. You can change this in the config file if you need. Anyway, For example: in order to change the SSH port
```
sudo vim /etc/ssh/sshd_config
```

Look for `Port` section and change from `22` to whatever port your want. And restart SSH server.
```
sudo systemctl restart sshd
sudo systemctl restart ssh
```

Then go to the sorcia directory which is under the git user.
```
cd /home/git/sorcia
```

Now, let's start the sorcia server.
```
sudo ./sorcia web
```

That's it, sorcia will run on port `1937`.

**Systemd, Nginx and Let's Encrypt configuration**

If you want to move further and setup your systemd service with your domain configured with Nginx, please follow
```
sudo cp /home/git/sorcia/config/sorcia-web.service /etc/systemd/system/
sudo systemctl start sorcia-web.service
sudo systemctl enable sorcia-web.service
```

Let's configure Nginx now. **Note:** Change the `git.example.com` to your domain address.
```
sudo mv /etc/nginx/sites-available/default /etc/nginx/sites-available/default.backup
sudo rm /etc/nginx/sites-enabled/default
sudo cp /home/git/sorcia/config/nginx.conf /etc/nginx/sites-available/git.example.com
sudo ln -s /etc/nginx/sites-available/git.example.com /etc/nginx/sites-enabled/
```

Now open the Nginx config file as shown below in order to mention your domain address.
```
sudo vim /etc/nginx/sites-available/git.example.com
```
and change the `git.example.com` to your domain address.

Now check if there is any problem with your Nginx config by
```
sudo nginx -t
```

if your Nginx config is correct. It will show you:
```
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
```

Reload the Nginx service.
```
sudo systemctl reload nginx
```

Now you can go to your domain and see the Sorcia software running. AND if you want to configure `https` certificate with Let's Encrypt, follow below commands.

Change the `git.example.com` to your domain address in the below certbot command. Follow the Let's Encrypt prompt and obtain the certificate.
```
sudo add-apt-repository ppa:certbot/certbot
sudo apt install python-certbot-nginx
sudo certbot --nginx -d git.example.com
```

Once you had successfully obtained the Let's Encrypt certificate. You have to go to the Nginx config file and uncomment these lines. Again, change the `git.example.com` to your domain address.
```
# ssl on;
# ssl_certificate /etc/letsencrypt/live/git.example.com/fullchain.pem; # managed by Certbot
# ssl_certificate_key /etc/letsencrypt/live/git.example.com/privkey.pem; # managed by Certbot
# include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot
# ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem; # managed by Certbot
```

And check and reload Nginx.
```
sudo nginx -t
sudo systemctl reload nginx
```

You can now see your domain with https served by Let's Encrypt.

### post installation
There is this important CLI utility from Sorcia which I need to mention. As Sorcia doesn't rely on SMTP settings, this CLI utility can be used to change:

 * Username of any user
 * Email address of any user
 * Password of any user
 * Delete any user. If it is an admin user you want to delete, the prompt will ask you to select another user as an admin before deletion of the current admin.

Remember, this can only be done by the server administrator who can SSH into the sorcia instance.

With the root user or user with root privileges, do `cd /home/git/sorcia` and enter the following command
```
sudo ./sorcia usermod
```

The command will prompt you for each of those above lists and by selecting one and following the further prompts, you can do those changes.