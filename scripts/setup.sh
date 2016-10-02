#!/bin/bash

echo "Preparing for hackathon.."

echo "Updating GOPATH..."
echo 'export GOPATH=$HOME/Workspace' >> ~/.bash_profile
echo 'export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin' >> ~/.bash_profile
source ~/.bash_profile

echo "Updating system"
sudo apt-get update && sudo apt-get upgrade -y

echo "Installing packages"
sudo apt-get install -y git vim htop golang apt-transport-https ca-certificates nodejs npm

go get -u github.com/govend/govend
sudo ln -s `which nodejs` /usr/bin/node

mkdir ~/Workspace
git clone https://github.com/timblankers/hyperledger-hackathon.git ~/Workspace/hyperledger-hackathon

sudo apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D

echo "deb https://apt.dockerproject.org/repo ubuntu-xenial main" | sudo tee /etc/apt/sources.list.d/docker.list

sudo apt-get update
sudo apt-get install -y docker-engine
sudo usermod -aG docker $(whoami)

curl -L https://github.com/docker/compose/releases/download/1.8.0/docker-compose-`uname -s`-`uname -m` > docker-compose
sudo mv docker-compose /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

exit
