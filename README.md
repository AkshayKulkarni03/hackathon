# Hyperledger Hackathon
## Introduction
Dear hackers, welcome to the first Hyperledger Hackathon. Are you up for the challenge? Just 36 hours. Make it fast. Make it awesome.

The wiki of this repository helps you to get going and provides all sort of other information:

[GENERAL INFORMATION](https://github.com/timblankers/hyperledger-hackathon/wiki)

## Getting Started
This rest of this readme documents the setup of a local blockchain. It connects a simple NodeJS backend to a Hyperledger Fabric peer. Deploying chaincode etc.

### System Requirements
Operating Systems:
* macOS Yosemite 10.10.3 or higher
* Windows 10 or higher
* Ubuntu 14.04 or higher

Older operating systems might work. See [Docker Toolbox](https://www.docker.com/products/docker-toolbox)

### Prerequisites
* git
* golang
* docker
* docker-compose
* nodejs
* npm
* govend

To install these packages on fresh install of Ubuntu 16.04.1, you can use [this setup script](scripts/setup.sh). After installation you have to log out and in of Ubuntu again.

### Installation
This installation procedure __assumes you are running Ubuntu 16.04.1__. You may use this as a reference if a different OS is used.

*HINT: You can install Ubuntu in Virtualbox. Allocate at least 16GB for the virtual machine. After that, make sure you install the Guest Additions and increase the video memory for better performance.*

Add new GOPATH and PATH variables to `~/.bashrc`
```
# Add these lines at the end of the file ~/.bashrc
export GOPATH=$HOME/Workspace
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
```
Reload the PATH
```
$ source ~/.bashrc
```
From the project directory `~/Workspace/hyperledger-hackathon`:
```
$ bash scripts/docker.sh 0.0.11
$ npm install
```

### Usage
```
$ ./start.sh
```

### Next Steps
If everything is up and running and working correctly you should be able to see a response from the peer at:

[localhost:7050/chain](http://localhost:7050/chain)

Simple response from your NodeJS backend at:

[localhost:3000](http://localhost:3000)

Query chaincode response at:

[localhost:3000/query](http://localhost:3000/query)

Invoke chaincode response at:

[localhost:3000/invoke](http://localhost:3000/invoke)

Your application can interact with the blockchain through an API, which is explained in the [NodeSDK Setup](http://hyperledger-fabric.readthedocs.io/en/latest/Setup/NodeSDK-setup/)

## License
MIT
