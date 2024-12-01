#!/bin/bash
export TERM=ansi
cd /home/bbs/git/mg_advent
./advent --path /talisman/temp/$1/ --debug-disable-date --debug-date=2024-12-25
exit

