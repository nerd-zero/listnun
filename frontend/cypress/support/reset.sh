#!/bin/bash

pkill -9 listnun
 cd ../
./listnun --install --yes
./listnun > /dev/null 2>/dev/null &
