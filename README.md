# unifi-r53
Reverse DNS for Unifi routers and AWS Route 53 hosted zones. This workflow would be unnecessary if Ubiquiti's built-in DDNS (inadyn) allowed users to allow unsecure http for the target server. If they did this could be a simple web server that recieved requests from the router and updated the r53 records from there. Or they could make this whole thing pointless by including r53 support themselves. It's not like AWS is some small obscure hosting platform or anything...

## TODOs
- Dockerize
- Run as a cron every 5 or 10 minutes
