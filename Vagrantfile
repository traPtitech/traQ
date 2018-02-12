Vagrant.configure("2") do |config|
	config.vm.box = "archlinux/archlinux"

	config.vm.network "forwarded_port", guest: 3000, host: 3000

	config.vm.provision :itamae do |config|
		config.sudo = true
		config.recipes = "./VagrantProvision.rb"
	end
end
