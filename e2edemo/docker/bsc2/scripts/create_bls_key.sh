#!/usr/bin/expect

set timeout 5
spawn geth bls account new --datadir [lindex $argv 0]
expect "*assword:*"
send "$env(BLS_WAL_PW)\r"
expect "*assword:*"
send "$env(BLS_WAL_PW)\r"
expect "*assword:*"
send "$env(BLS_ACC_PW)\r"
expect "*assword:*"
send "$env(BLS_ACC_PW)\r"
expect EOF
