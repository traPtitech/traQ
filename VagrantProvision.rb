execute "pacman -Sy"

execute "hostnamectl set-hostname traQ-dev.local" do
	not_if "hostnamectl | grep traQ-dev.local"
end

["go", "git", "make"].each do |pkg|
	package pkg
end

projectDir = "/home/vagrant/go/src/github.com/traPtitech"
directory projectDir do
	user "vagrant"
end

link "#{projectDir}/traQ" do
	user "vagrant"
	to "/vagrant"
end

file "/home/vagrant/.bashrc" do
	content <<~EOS
		export GOPATH=/home/vagrant/go #
		export PATH=$PATH:$GOPATH/bin #
		export EDITOR=nano #
		cd #{projectDir}/traQ #
	EOS
end
