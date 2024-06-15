# Raspberry Pi (Linux) network reconnector



automatically reconnect when the Raspberry Pi loses network connection



## Build

### windows

```batch
.\scripts\build.bat
```

### linux

```bash
./scripts/build.sh
```



## Run

put `./scripts/network-reconnector.service` to `/etc/systemd/system/`



```bash
sudo systemctl enable network-reconnector.service
sudo systemctl restart network-reconnector.service
```