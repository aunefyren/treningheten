#!/bin/sh

# Build the argument list with positional parameters so every value stays one quoted
# argument. The dbpassword and smtppassword environment variables are read by the binary
# directly and deliberately not passed as flags (flags are visible in the process list).
set -- /app/treningheten

[ -n "$port" ] && set -- "$@" --port "$port"
[ -n "$externalurl" ] && set -- "$@" --externalurl "$externalurl"
[ -n "$timezone" ] && set -- "$@" --timezone "$timezone"

[ -n "$dbip" ] && set -- "$@" --dbip "$dbip"
[ -n "$dbport" ] && set -- "$@" --dbport "$dbport"
[ -n "$dbname" ] && set -- "$@" --dbname "$dbname"
[ -n "$dbusername" ] && set -- "$@" --dbusername "$dbusername"

[ -n "$generateinvite" ] && set -- "$@" --generateinvite "$generateinvite"
[ -n "$disablesmtp" ] && set -- "$@" --disablesmtp "$disablesmtp"
[ -n "$smtphost" ] && set -- "$@" --smtphost "$smtphost"
[ -n "$smtpport" ] && set -- "$@" --smtpport "$smtpport"
[ -n "$smtpusername" ] && set -- "$@" --smtpusername "$smtpusername"
[ -n "$smtpfrom" ] && set -- "$@" --smtpfrom "$smtpfrom"

exec "$@"
