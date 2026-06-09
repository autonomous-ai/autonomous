sudo nmcli connection add type wifi ifname wlan0 con-name Glinks ssid "Glinks"
sudo nmcli connection modify Glinks wifi-sec.key-mgmt wpa-psk
sudo nmcli connection modify Glinks wifi-sec.psk "@11235813"
sudo nmcli connection up Glinks