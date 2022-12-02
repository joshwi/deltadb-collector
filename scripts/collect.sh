#!/bin/bash
while getopts "y:" arg 
do
  case $arg in
    y) year=$OPTARG;;
  esac
done

if [ -z ${year+x} ]; 
then         
    echo "variable is unset: year" >&2
    exit 1; 
else 
    $BASE_PATH/app/builds/transactions -file="config/import/static.json"
    $BASE_PATH/app/builds/collector -name="teams"
    $BASE_PATH/app/builds/transactions -file="config/aggregate/teams.json"
    $BASE_PATH/app/builds/collector -name="picks" -query="MATCH (n:years) WHERE n.year='$year' RETURN n.year as year"
    $BASE_PATH/app/builds/collector -name="seasons" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
    $BASE_PATH/app/builds/collector -name="nfl_games" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
    $BASE_PATH/app/builds/transactions -file="config/aggregate/games.json"
    $BASE_PATH/app/builds/collector -name="stat_player_game" -query="MATCH (n:games) WHERE n.year='$year' RETURN DISTINCT n.game_code as game_code"
    # $BASE_PATH/app/builds/export -filepath="deltadb-archive/nfl/$year" -query="WHERE n.year='$year'" -nodes="stat.*|games|picks|seasons"
fi