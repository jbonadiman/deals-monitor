# Deals monitor

I created this Vercel serverless Go function with the intention of monitoring deals across several sources, and I started by Telegram public channels (which is the only available source for now). The function matches the texts of deals using regex and notifying whatever it finds using [Pushover](https://pushover.net/), which is of course, needed for this to work. It also uses a Redis instance to keep track of the deals it already parsed during the day.

Backlog:
- use a embeddable go database instead of redis to reduce dependencies (badger?)
- notify through other services
- grab deals from other sources
