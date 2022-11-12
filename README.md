sudo docker build -t collector . --no-cache --progress=plain 
sudo docker run -d --env-file .env --name collector collector