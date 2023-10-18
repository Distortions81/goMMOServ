Use a real cert, or letsencrypt.
privkey.pem
fullchain.pem

devtest only:
openssl genrsa -out server.key 2048
openssl ecparam -genkey -name secp384r1 -out privkey.pem
openssl req -new -x509 -sha256 -key privkey.pem -out fullchain.pem -days 3650

Testing:
curl -X POST -H "Content-Type: application/json" -d 'CheckUpdateDev:v018-2023-05-02-05-23-20' https://localhost:8648 -k


GONOPROXY=github.com/Distortions81/namegenerator go get -d github.com/Distortions81/namegenerator