#! /usr/bin/env bash


OS=""
GVM=false
NOPROMPT=false

function detectGVM {
	if command -v gvm >/dev/null 2>&1; then 
		GVM=true
	fi
}

function installGVM {
	echo "Installing GVM (go version manager).";
	if [ "$OS" == "DEB" ]; then
		if sudo apt-get install -y curl git mercurial make binutils bison gcc build-essential; then
			# Install succeeded.
		else
			echo "Could not install required debian packages.";
			echo "Please install these packages manually and run this script again:";
			echo "apt-get install curl git mercurial make binutils bison gcc build-essential";
			exit 1;
		fi
	elif [ "$OS" == "REDHAT" ];
		if sudo yum install -y curl git make bison gcc glibc-devel; then
			# Install succeeded.
		else
			echo "Could not install required yum packages.";
			echo "Please install these packages manually and run this script again:";
			echo "yum install curl git make bison gcc glibc-devel";
			exit 1;
		fi
	fi
	elif [ "$OS" == "OSX" ]; then
		echo "OS X detected";
		echo "Please install install Mercurial from http://mercurial.berkwood.com/"
		echo "and install Xcode Command Line Tools from the App Store."
		if $NOPROMPT; then
			# Assume that it's already installed.
		else
			echo "Do you have Mercurial and Xcode installed?";
			select flag in "y" "n"; do
				case $flag in 
					y ) break;;
					n ) echo "Install them and run this script again"; exit1;;
				esac
			done
		fi
	fi

	# Dependencies installed, try to install GVM.
	bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
}

function determineOS {
	# Detect debian based systems.
	if command -v apt-get >/dev/null 2>&1; then
		OS="DEB";
		return;
	fi

	# Detect redhat/centos based systems.
	if command -v yum >/dev/null 2>&1; then
		OS="REDHAT";
		return;
	fi

	# Detect OS X.
	if [ "$OSTYPE" == "darwin"* ]; then
		OS="OSX";
		return;	
	elif [ "$OSTYPE" == "linux-gnu" ]; then
		# Generic linux.
		OS="LINUX";
		return;
	fi
}

function installGO {
	echo "Installing GO";

	# Install go1.4.
	if gvm install go1.4; then
	else 
		echo "Could not install go1.4";
		exit 1
	fi
	if gvm use go1.4; then
	else
		echo "Could not switch to go1.4";
		exit 1
	fi

	# Install go1.5.
	if gvm install go1.5; then
	else 
		echo "Could not install go1.5";
		exit 1
	fi
	if gvm use go1.5; then
	else
		echo "Could not switch to go1.5";
		exit 1
	fi
}

function installAppkit {
	echo "Installing appkit + and its dependencies.";
	export GOPATH="~/.go_appkit"
	mkdir $GOPATH
	if go get github.com/app-kit/go-appkitcli; then
	else
		echo "Could not install appkit or one of it's dependencies.";
		exit 1
	fi

	if go install github.com/app-kit/go-appkitcli/appkit; then
		echo "Could not install appkit command.";
		exit 1
	fi
}

function install {
	detectGVM
	determineOS

	# If os could not be detected, gvm must be installed manually.
	if [ -z "$OS" ]; then
		# Check if GVM is already installed.
		if [ $GVM = false ]; then
			echo "Could not detect the OS. Please manually install GVM (https://github.com/moovweb/gvm).";
			echo "Then run this script again.";
			exit 1
		fi
	fi

	installGVM
	installGO
	installAppkit

	echo "To use the appkit command in any session, add this to your shell config:"
	echo ""

	echo "Appkit has been successfully installed. Start a new project with 'appkit bootstrap ...'"
}
