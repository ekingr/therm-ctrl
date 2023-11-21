Thermostat control server
=========================
2022, (c) ekingr


Server running on a raspberry pi zero, controlling the thermostat hardware (via MCP23S17).
Offering a simple API to read sensors status and set relays state.


TODO
----

- [ ] Restrict incoming ip address to proxy server (192.168.xxx.xxx)
- [ ] Handle case when MSP is not connected
- [ ] Log interrupts
- [ ] Implement a constant-time validation of API KEY (see https://pkg.go.dev/crypto/subtle)

- [x] Inverse sensors status
- [x] Throttle when several relays need to change at the same time
- [x] Throttling relay state change
- [x] Build a time-out to disable relays when no GET for some time (eg. loss of connectivity)
- [x] Setup rpi properly (out of this project's scope, but has to be done)
- [x] daemon configuration
- [X] Start daemon after wireguard
- [x] Restart wireguard periodically


Architecture
------------

- *Hardware brain*: MCP23S17 interfacing with the 3 boards, controlling the relays and reading the sensors
- SPI interface on the GPIO
- *Control server*: raspberry pi zero offering REST API to control / read the hardware brain
- REST API over HTTPS over Wireguard VPN tunnel
- *Gateway server*: raspberry pi 4 server, internet-facing, with HTTPS termination and advanced access control
- HTTPS API
- *Client*: HTML/CSS/JS UI


Features
--------

### Safety
- Watchdog reverts to safe state (all relays off) if no request for 3 hours (internet connection issue, ui server issue...)
- Throttling of state change requests to 1 per second maximum
- Changing relays state one at a time with 200ms wait in between to avoid current / voltage surge

### Security
- Connected through VPN (wireguard), with only access to UI server and dev box
- HTTPS API to encrypt interactions (with self-signed key)
- API key authenticates all interactions (GET & POST)


Raspberry pi zero configuration
-------------------------------

### Pre-boot OS configuration ###

Since the install will be headless, some modifs need to be made to the SD card before the first boot.
Plug the card in again and open the disk manager ("Gestion de diques") and assign a drive letter to the 256MB `boot` partition.

Add an empty `ssh` file to the root of the `boot` partition to allow remote ssh connection for setup.

To create a new user -- which is needed to be able to connect via SSH -- add a `userconf` file:
```
username:encryptedPassword
```
where `username` is the user name to be created, and `encryptedPassword` is the password encrypted with openssl: `echo 'mypassword' | openssl passwd -6 -stdin`

If wifi is needed to configure the Rpi, add a `wpa_supplicant.conf` file:
```
update_config=1
country=FR
network={
    ssid=""
    psk=""
    key_mgmt=WPA-PSK
}
```

Eject the card.
NB: if the Rpi is booted then those modifications disapead and need to be done again.

Plug the card into the Rpi and wire all the required components. Plug in the power last to boot up the Rpi.
See the router admin interface to find out the ip address of the Rpi.
Ssh into it (eg. with `putty` if on windows or `ssh` on unix).

### create user ###

If no specific user was created (using `pi` as in older distros), create a new one:
```shell
$ sudo less /etc/passwd
$ groups pi
$ sudo adduser xxx
$ groups xxx
$ sudo usermod -aG sudo xxx
$ sudo usermod -aG adm xxx
$ sudo usermod -aG dialout xxx
$ sudo usermod -aG cdrom xxx
$ sudo usermod -aG audio xxx
$ sudo usermod -aG video xxx
$ sudo usermod -aG plugdev xxx
$ sudo usermod -aG games xxx
$ sudo usermod -aG users xxx
$ sudo usermod -aG input xxx
$ sudo usermod -aG netdev xxx
$ sudo usermod -aG spi xxx
$ sudo usermod -aG i2c xxx
$ sudo usermod -aG gpio xxx
```

Manage what can be run with `sudo` without a password:
```shell
$ sudo vim /etc/sudoers.d/010_xxx-nopasswd
xxx ALL = NOPASSWD: /bin/kill, /usr/sbin/service, /usr/bin/htop

$ sudo vim /etc/sudoers.d/010_pi-nopasswd
# pi ALL=(ALL) NOPASSWD: ALL
```

Later-on ssh access to `pi` can also be revoked, so as login can only be through `su` or physically.

### raspi-config ###

Run `sudo raspi-config` to do the primary configuration:
- 5 Localisation
    - L1 Locale: `en_GB UTF-8`
    - L2 Timezone: `Europe`, `Paris`
    - L3 Keyboard
    - L4 WLAN country: `FR`
- 1 System
    - S3 Password
    - S4 Hostname: `therm`
    - S5 Boot: B1 console
    - S6 Network at boot: no
- 3 Interface
    - I1 Legacy camera: no
    - I2 SSH: yes
    - I3 VNC: no
    - I4 SPI: yes
    - I5 I2C: no
    - I6 Serial: no
    - I7 1-wire: no
    - I8 Remote-GPIO: no
- 4 Performance
    - P2 GPU memory: 16MB
- 6 Advanced
    - A1 Expand filesystem

Reboot.

### Basic software ###

Update and install basic software
```shell
$ sudo apt update
$ sudo apt upgrade
$ sudo apt install htop
$ sudo apt install vim
```

### Hardware configuration ###

Disable Bluetooth
Add to the boot config file:
```shell
$ sudo vim /boot/config.txt
# Disable Bluetooth:
dtoverlay=disable-bt
```

Disable the Bluetooth & Bluetooth audio services
```shell
$ sudo systemctl disable bluealsa.service
$ sudo systemctl disable bluetooth.service
```

Disable the Modem service
```shell
$ sudo systemctl disable hciuart.service
```

Disable the Avahi service
```shell
$ sudo systemctl disable avahi-daemon.service
```

Disable Audio
Modify the `dtparam=audio` parameter in the boot config file:
```shell
$ sudo vim /boot/config.txt
#was: dtparam=audio=on
dtparam=audio=off
```

Disable HDMI (may not work anymore on newer revisions of the OS)
```shell
$ sudo vim /etc/rc.local
# Disable HDMI
/usr/bin/tvservice -o
```

Disable IPV6
Create 2 files:
```shell
$ sudo vim /etc/sysctl.d/disable-ipv6.conf
net.ipv6.conf.all.disable_ipv6 = 1

$ sudo vim /etc/modprobe.d/blacklist-ipv6.conf
blacklist ipv6
```

Set static IP
Make sure the static IP is compatible with the gateway DHCP ranges.
It can be a good idea to configure a similar static lease on the gateway just in case.
Edit the `dhcpcd` config file to add the static address and DNS servers:
```shell
$ sudo vim /etc/dhcpcd.conf
# static IP configuration:
interface wlan0
static ip_address=192.168.xxx.xxx/24
static routers=192.168.xxx.xxx
static domain_name_servers=1.1.1.1 1.0.0.1 8.8.8.8 8.8.4.4
```

Reboot.

### Vim ###

Edit config:
```shell
$ vim ~/.vimrc
syntax enable

set t_Co=256
set background=dark
colorscheme gruvbox

set tabstop=4
set softtabstop=4
set shiftwidth=4
set expandtab
filetype indent plugin on
set autoindent

set number
set cursorline
set wildmenu

set showmatch
set incsearch
set hlsearch

```

Set as default editor:
```shell
$ vim .profile
export EDITOR="/usr/bin/vim"
```

### Firewall: UFW ###

Install and set basic firewall rule:
```shell
$ sudo apt install ufw
$ sudo ufw allow ssh
$ sudo ufw enable
$ sudo ufw status
```

### SSH server: SSHd ###

SSHd should be already installed and running.

Improve configuration:
```shell
$ sudo vim /etc/ssh/sshd_config
> LoginGraceTime 30
> permitRootLogin no
> StrictModes yes
> AllowUsers xxx xxx
> PubkeyAuthentication yes
> PermitEmptyPasswords no
```

Copy the local ssh key over to the server:
```shell
(dev)$ ssh-copy-id xxx@yyy
```

Disable password authentication:
```shell
$ sudo vim /etc/ssh/sshd_config
> PasswordAuthentication no

$ sudo service ssh reload
```

### Wireguard VPN ###

Install Wireguard:
```shell
$ sudo apt install wireguard
```

Check installation is working properly:
```shell
$ # Check binary installation
$ which wg wg-quick
/usr/bin/wg
/usr/bin/wg-quick

$ # Set-up dummy configuration
$ sudo touch /etc/wireguard/wg0.conf

$ # Loading & starting dummy configuration
$ wg-quick up wg0
[#] ip link add wg0 type wireguard
[#] wg setconf wg0 /dev/fd/63
[#] ip link set mtu 1420 up dev wg0

$ # Checking kernel module is properly loaded
$ lsmod | grep wire
wireguard              69632  0
libchacha20poly1305    16384  1 wireguard
ip6_udp_tunnel         16384  1 wireguard
udp_tunnel             28672  1 wireguard
libcurve25519_generic    40960  1 wireguard
libblake2s             16384  1 wireguard
ipv6                  552960  26 nf_reject_ipv6,wireguard

$ # Checking interfaces
$ ifconfig wg0
wg0: flags=209<UP,POINTOPOINT,RUNNING,NOARP>  mtu 1420
        unspec 00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00  txqueuelen 1000  (UNSPEC)
        RX packets 0  bytes 0 (0.0 B)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 0  bytes 0 (0.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

$ sudo wg
interface: wg0
  listening port: 52595

$ # If everything ok up to there, bringing down the dummy interface
$ wg-quick down wg0
$ sudo wg
$ ifconfig wg0
$ lsmod | grep wire
$ sudo modprobe -r wireguard
$ lsmod | grep wire
```

Generate keys:
```shell
$ mkdir wg_config
$ cd wg_config
$ # Using umask to set file creation permissions
$ umask 077
$ wg genkey > xxxxx_private.key
$ wg pubkey > xxxxx_public.key < xxxxx_private.key
```

Edit the config:
```shell
$ sudo vim /etc/wireguard/wg0.conf
[Interface]
Address = 192.168.xxx.xxx/24
PrivateKey = [value in xxxxx_private.key]
DNS = 1.1.1.1, 1.0.0.1, 8.8.8.8, 8.8.4.4

[Peer]
PublicKey = [value of vpn server public key]
# Only allow to access server (gateway server) and dev box (for ssh manual operations)
AllowedIPs = 192.168.xxx.xxx/32, 192.168.xxx.xxx/32
Endpoint = xxx:yyy
# If needs to keep connction chatty to maintain NAT transversal (send a packet every N seconds):
PersistentKeepalive = 29
```

Checking configuration:
```shell
$ sudo wg-quick up wg0
$ sudo wg
  public key: XXXX
  private key: (hidden)
  listening port: xxx

peer: XXX
  endpoint: 109.192.xxx.xxx
  allowed ips: 192.168.xxx.xxx/32, 192.168.xxx.xxx/32
  latest handshake: 53 seconds ago
  transfer: 26.22 KiB received, 57.75 KiB sent
  persistent keepalive: every 29 seconds


$ sudo wg-quick down wg0
```

Launch wireguard:
```shell
$ sudo systemctl enable wg-quick@wg0
$ sudo systemctl status wg-quick@wg0
$ sudo wg
```

For good measure, hostnames for the relevant vpn addresses can be added:
```shell
$ sudo vim /etc/hosts
# Wireguard VPN
192.168.xxx.xxx  srv.my.example.com
192.168.xxx.xxx  ctl.my.exampl.com
```

Adding cron job to restart wireguard service every night
(fix for when things did not start properly, eg. when rebooting at same time as the internet router after a power outage)
```shell
$ # Command that will be run every day at 04:00
$ sudo service wg-quick@wg0 restart
$ # Check that  wg-quick@wg0 is the right service name
$ # Check actual binary of service command
$ sudo which service
$ # Edit crontab for root user
$ sudo crontab -e

# Wireguard automatic restart every day at 04:00
0 4 * * * /usr/sbin/service wg-quick@wg0 restart
```

### TLS certificates creation ###

Create RSA certificate for the HTTPS encryption:
```shell
$ openssl req -x509 -nodes -sha256 -days 3650 -newkey rsa:2048 -keyout privkey.pem -out fullchain.pem -subj "/C=FR/ST=Paris/L=Paris/O=xxx/OU=xxx/CN=ctl.my.example.com/emailAddress=postmaster@my.example.com"
$ openssl x509 -text -in fullchain.pem -noout
$ chmod 600 fullchain.pem
$ chmod 600 privkey.pem
```

```shell
$ sudo ufw allow 9443
$ sudo ufw enable
$ sudo ufw status
```


### Journalctl: Systemd logging ###

If running systemd, configure logging by adding:
```shell
$ # man journalctl
$ # man journald.conf
$ sudo vim /etc/systemd/journald.conf

# Keep logs on disk at /var/log/journal
Storage=persistent
# Compress journal objects
Compress=yes
# Size limit on journal stored on filesystem:
# Use 20% of the filesystem size at most
SystemMaxUse=20%
# Keep at least 20% of free filesystem space
SystemKeepFree=20%
```

