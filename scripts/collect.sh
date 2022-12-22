#!/bin/bash
cd "$(dirname "$1")"
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
    app/builds/transactions -file="deltadb-assets/transactions/import/static.json"
    app/builds/collector -name="teams"
    app/builds/transactions -file="deltadb-assets/transactions/aggregate/teams.json"
    app/builds/collector -name="picks" -query="MATCH (n:years) WHERE n.year='$year' RETURN n.year as year"
    app/builds/collector -name="seasons" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
    app/builds/collector -name="nfl_games" -query="MATCH (n:teams) RETURN DISTINCT n.tag as tag, '$year' as year"
    app/builds/transactions -file="deltadb-assets/transactions/aggregate/games.json"
    app/builds/collector -name="stat_player_game" -query="MATCH (n:games) WHERE n.year='$year' RETURN DISTINCT n.game_code as game_code"
    app/builds/transactions -file="deltadb-assets/transactions/aggregate/stats.json"
    # app/builds/transactions -file="deltadb-assets/transactions/relationships/stats.json"
    # app/builds/export -filepath="deltadb-archive/nfl/$year" -query="WHERE n.year='$year'" -nodes="stat.*|games|picks|seasons"
fi