Prosu for Twitter
=================
The full rewrite of [Prosu](https://github.com/wcalandro/prosu)  
This is live at [https://prosu.xyz](https://prosu.xyz)

I learned Go during the summer, and was inspired to rewrite Prosu as my summer project.  
The end result is a website that runs faster with only half the RAM usage. (~125MB avg -> ~50MB avg)

Requirements
------------
To run Prosu, you will need two different types of databases:
- MongoDB (for persistent user data)
- Redis (for temporary user sessions)  

Additionally, you will need a variety of environment variables:

| Environment Variable | Description                                                  | Required                           |
|----------------------|--------------------------------------------------------------|------------------------------------|
| `OSU_API_KEY`        | API key from [osu!](https://osu.ppy.sh/p/api)                | Yes                                |
| `ENVIRONMENT`        | The environment to run the application in (eg. "production") | No (default: development)          |
| `DOMAIN`             | The domain the website will be accessed on                   | Yes                                |
| `CONSUMER_SECRET`    | Twitter Consumer Secret Token                                | Yes                                |
| `CONSUMER_KEY`       | Twitter Consumer Public Token                                | Yes                                |
| `REDIS_HOST`         | Host and port for redis server (eg localhost:6379)           | Yes                                |
| `REDIS_PASSWORD`     | Password for redis login                                     | Yes                                |
| `SESSION_SECRET`     | Secret key to encrypt cookies                                | Yes                                |
| `NEWRELIC_KEY`       | NewRelic License key to send info to NewRelic                | Yes if environment is "production" |
| `ROLLBAR_TOKEN`      | Rollbar API token to send errors to Rollbar                  | Yes if environment is "production" |
| `MONGO_URL`          | MongoDB connection URI                                       | Yes                                |
| `PORT`               | Port to listen on                                            | No (default: 5000)                 |

Dependencies
------------
The project's dependencies were vendored with [dep](https://github.com/golang/dep). If you have dep installed you can run:
```
dep ensure
```
to install the project's dependencies

Running Prosu for Twitter
-------------------------
You can run Prosu for Twitter anywhere, however it was intended to be run on [Heroku](https://heroku.com) or [Dokku](https://github.com/dokku/dokku), and includes a `Dockerfile` that works with both.