# example

fabric-ca-client enroll -d -u http://admin:adminpw@localhost:7054 --mspdir ca-admin

fabric-ca-client register -d --id.name org1peer0 --id.secret org1peer0pw -u http://admin:adminpw@localhost:7054 --id.type peer --mspdir ca-admin
fabric-ca-client enroll -d -u http://org1peer0:org1peer0pw@localhost:7054 --mspdir peer0

fabric-ca-client register -d --id.name org1peer1 --id.secret org1peer1pw -u http://admin:adminpw@localhost:7054 --id.type peer --mspdir ca-admin
fabric-ca-client enroll -d -u http://org1peer1:org1peer1pw@localhost:7054 --mspdir peer1

fabric-ca-client register -d --id.name org1admin --id.secret org1adminpw -u http://admin:adminpw@localhost:7054 --id.type admin --mspdir ca-admin
fabric-ca-client enroll -d -u http://org1admin:org1adminpw@localhost:7054 --mspdir admin

fabric-ca-client register -d --id.name org1client --id.secret org1clientpw -u http://admin:adminpw@localhost:7054 --id.type client --mspdir ca-admin
fabric-ca-client enroll -d -u http://org1client:org1clientpw@localhost:7054 --mspdir client
