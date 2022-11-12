year=$(date +%Y)
collector=/deltadb/config/collector.json
/deltadb/app/builds/transactions -file="/deltadb/config/import/static.json"
/deltadb/app/builds/collector -file="$collector" -name="teams"
/deltadb/app/builds/collector -file="$collector" -name="picks" -query="MATCH (n:years) WHERE n.year='$year' RETURN n.year as year"
/deltadb/app/builds/collector -file="$collector" -name="seasons" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
/deltadb/app/builds/collector -file="$collector" -name="nfl_games" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
/deltadb/app/builds/transactions -file="/deltadb/config/aggregate/games.json"
/deltadb/app/builds/collector  -file="$collector" -name="stat_player_game" -query="MATCH (n:games) WHERE n.year='$year' RETURN DISTINCT n.game_code as game_code"
