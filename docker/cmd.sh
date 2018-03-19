if [[ -f $CONFIG_FILE ]]; then
    /bin/promteams server -p 2000 --config $CONFIG_FILE
else
    /bin/promteams server -p 2000 -w "${TEAMS_INCOMING_WEBHOOK_URL}" -r "${TEAMS_REQUEST_URI}"
fi