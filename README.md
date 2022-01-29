WebFive SNI Proxy

You can use this app for forwarding web requests to a socks5 server... It's some kind of http reverse proxy. It's simply
a web server like nginx witch has reverse proxy who forwards traffic to socks5 instead of another web server.

Requirements:

1. A Socks5 Server (Like Shadowsocks...).
2. A DNS Server (Like AdGuardHome...).
3. WebFive SNI Proxy.

How to use:

1. First configure a socks5 server, and it's client, usually server must be in outside and client must be in the inside
   of network.
2. Then config WebFive SNI Proxy, Edit the default `config.yml` or create a new one.
3. If create a new config file run it via `-C` command like `./webFiveSniProxy -C=mycfg`, don't add `.yml` file
   extension.
4. Otherwise, run it normally `./webFiveSniProxy`.
5. Then configure a DNS server inside network and add `A` records for the domains you want to forward all traffic from
   them to socks proxy.
6. These `A` or `AAAA` records must point to the WebFive SNI Proxy server you executed above.
7. Then in the same DNS server all rules to solve all other domains through a public dns server (like 8.8.8.8), because
   if you don't do this all other domains will not be solved and only the domains you configure will be working.
8. Then set this DNS server as your networks primary DNS server (static or dhcp).
9. Run DNS server and flush all DNS caches of your network and your systems.

Config file docs:

server: `This section is SNI Proxy Server listening configs, it's simple a web server like nginx.`<br/>
&nbsp;&nbsp;&nbsp;&nbsp;httpHost: "0.0.0.0"<br/>
&nbsp;&nbsp;&nbsp;&nbsp;httpPort: "80"<br/>
&nbsp;&nbsp;&nbsp;&nbsp;httpsHost: "0.0.0.0"<br/>
&nbsp;&nbsp;&nbsp;&nbsp;httpsPort: "443"<br/>
proxy: `This section is the backend of the Web Server you configured in Server section.`<br/>
&nbsp;&nbsp;&nbsp;&nbsp;socks5Host: "127.0.0.1"<br/>
&nbsp;&nbsp;&nbsp;&nbsp;socks5Port: "1080"<br/>

Usually you should keep the default config specially the `Server` section and config the DNS and Socks5 servers matching
it.