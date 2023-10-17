#!/bin/bash
git pull
go clean
go build
killall goMMOServ
