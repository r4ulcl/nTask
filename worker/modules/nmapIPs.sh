#!/bin/bash

scanRange=$1

nmap -sn "$scanRange" | grep -E -o "([0-9]{1,3}\.){3}[0-9]{1,3}"