# echo $BASE_PATH
year=$(date +%Y)
$BASE_PATH/app/builds/transactions -file="config/import/static.json"
$BASE_PATH/app/builds/collector -name="teams"
$BASE_PATH/app/builds/collector -name="picks" -query="MATCH (n:years) WHERE n.year='$year' RETURN n.year as year"
$BASE_PATH/app/builds/collector -name="seasons" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
$BASE_PATH/app/builds/collector -name="nfl_games" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
$BASE_PATH/app/builds/transactions -file="config/aggregate/games.json"
$BASE_PATH/app/builds/collector -name="stat_player_game" -query="MATCH (n:games) WHERE n.year='$year' RETURN DISTINCT n.game_code as game_code"
