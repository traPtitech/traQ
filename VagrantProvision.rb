goPath = "/home/vagrant/go"
projectDir = "#{goPath}/src/github.com/traPtitech"

execute "hostnamectl set-hostname traQ-dev.local" do
	not_if "hostnamectl | grep traQ-dev.local"
end

execute "pacman -Sy"
["go", "git", "make", "mariadb"].each do |pkg|
	package pkg
end

directory projectDir do
	user "vagrant"
end
link "#{projectDir}/traQ" do
	user "vagrant"
	to "/vagrant"
end

file "/home/vagrant/.bashrc" do
	content <<~EOS
		export EDITOR=nano #
		export GOPATH=#{goPath} #
		export PATH=$PATH:$GOPATH/bin #
		cd #{projectDir}/traQ #
	EOS
end

execute "Setup DB" do
	command <<~EOS
		mysql_install_db --user=mysql --basedir=/usr --datadir=/var/lib/mysql
		systemctl start mariadb
		mysql --user=root --execute='
			CREATE DATABASE `traq` CHARACTER SET = utf8mb4;
			CREATE DATABASE `traq-test-model` CHARACTER SET = utf8mb4;
			CREATE DATABASE `traq-test-router` CHARACTER SET = utf8mb4;
			SET PASSWORD = PASSWORD("password");
		'
	EOS
	not_if '[ "$(ls -A /var/lib/mysql)" ]'
end
service "mariadb" do
	action [:enable, :start]
end
