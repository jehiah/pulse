#
# Regular cron jobs for the tb-pulse package
#
0 4	* * *	root	[ -x /usr/bin/tb-pulse_maintenance ] && /usr/bin/tb-pulse_maintenance
