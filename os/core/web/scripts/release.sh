echo "========== Upload Built Application... =========="
sshpass -p "12345" ssh system@192.168.1.13 'rm -rf /usr/share/nginx/html/setup/*'
echo "========== Upload Assets... =========="
sshpass -p "12345" scp -r ./dist/assets system@192.168.1.13:/usr/share/nginx/html/setup
echo "========== Upload Index.html... =========="
sshpass -p "12345" scp ./dist/index.html system@192.168.1.13:/usr/share/nginx/html/setup
echo "========== Restart service =========="
sshpass -p "12345" ssh system@192.168.1.13 'sudo systemctl restart nginx'
