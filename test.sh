#! /bin/bash

echo "url1 $TEAMS_WEBHOOK_URL"
echo "url2 ${{ env.TEAMS_WEBHOOK_URL }}"
echo "url3 ${{ vars.TEAMS_WEBHOOK_URL }}"
