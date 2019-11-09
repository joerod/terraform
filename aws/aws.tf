provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  region     = "${var.aws_region}"
}

resource "aws_security_group" "ssh" {
  name = "Allow SSH"
  description = "Managed by JoeRod"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "ssh"
  }
}

resource "aws_instance" "joerod" {
    ami           = "ami-b374d5a5"
    instance_type = "t2.micro"
    key_name = "${aws_key_pair.joerod.key_name}"
    vpc_security_group_ids = ["${aws_security_group.ssh.id}"]
    provisioner "local-exec" {
    command = "Get-date | out-file -append ip_address.txt; ${aws_instance.joerod.public_ip} | out-file -append ip_address.txt"
    interpreter = ["PowerShell", "-Command"]
   
  }
   tags = {
       Name  = "${var.aws_resource_name}"
     }

}

resource "aws_eip" "ip" {
  instance = "${aws_instance.joerod.id}"
}