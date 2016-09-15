# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "ubuntu/trusty64"
  config.ssh.insert_key = false

  config.vm.define 'api-proxy' do |proxy|
    proxy.vm.network "private_network", ip: "192.168.10.4"
    proxy.vm.hostname = "proxy"
    proxy.vm.synced_folder ".", "/vagrant", disabled: true
    proxy.vm.synced_folder ".", "/home/vagrant/fr8r/", mount_options: ["dmode=775,fmode=775"]
    
    proxy.vm.provision "shell", inline: <<-KUBECTL
          echo "Installing kubectl..."
          kubectl_version=`curl -sL https://storage.googleapis.com/kubernetes-release/release/stable.txt`
          curl -sSLo /usr/local/bin/kubectl "https://storage.googleapis.com/kubernetes-release/release/$kubectl_version/bin/linux/amd64/kubectl"
          chmod +x /usr/local/bin/kubectl
    KUBECTL

    proxy.vm.provision "docker", images: [ "fr8r/api-proxy" ]
  end
end