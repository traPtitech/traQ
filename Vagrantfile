Vagrant.configure("2") do |config|
	config.vm.box = "archlinux/archlinux"

	config.vm.network "forwarded_port", guest: 9000, host: 9000

	config.vm.provision :itamae do |config|
		config.sudo = true
		config.recipes = "./VagrantProvision.rb"
	end
end
