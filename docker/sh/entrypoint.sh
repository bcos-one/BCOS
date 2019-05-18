#!/usr/bin/env sh
# $0 is a script name,
# $1, $2, $3 etc are passed arguments
# $1 is our command
CMD=$1

case "$CMD" in
 "init" )
  exec sh /bin/init.sh
  ;;

 "start" )
  exec sh /bin/start.sh
  ;;

 "stop" )
  exec sh /bin/stop.sh
  ;;

  * )
  # Run custom command. Thanks to this line we can still use
  # "docker run our_image /bin/bash" and it will work
  exec $CMD ${@:2}
  ;;
esac