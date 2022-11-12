year=$(date +%Y)
collector=config/collector.json
./app/builds/transactions -file="config/import/static.json"
./app/builds/collector -file="$collector" -name="teams"
./app/builds/collector -file="$collector" -name="picks" -query="MATCH (n:years) WHERE n.year='$year' RETURN n.year as year"
./app/builds/collector -file="$collector" -name="seasons" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
./app/builds/collector -file="$collector" -name="nfl_games" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
./app/builds/transactions -file="config/aggregate/games.json"
./app/builds/collector  -file="$collector" -name="stat_player_game" -query="MATCH (n:games) WHERE n.year='$year' RETURN DISTINCT n.game_code as game_code"
