Setting up the alerts to be shown on Telegram consists of a couple of steps:
1. Create a channel on Telegram
2. Create a bot with @botfather
3. Add the bot to the newly created channel and make it an admin
4. Add the Telegram channel to the config.yml on your TenderDuty configuration

Let's do it step by step:<br />
#### 1. Create a channel on Telegram<br />
Adding a channel goes best by following the following guide:<br />
https://www.alphr.com/telegram-how-to-create-supergroup/<br />
NOTE: you don't need to add people to the group at this point in time.

#### 2. Create a bot with @botfather<br />
For the Telegram signals you need to create a bot. Use the following guide to do so:<br />
https://riptutorial.com/telegram-bot/example/25075/create-a-bot-with-the-botfather<br />
NOTE: save the API key from this bot, you need it later.<br />

#### 3. Add the bot to the newly created channel and make it an admin<br />
To add the bot to the Telegram group you created in the first step follow this guide:<br />
https://www.alphr.com/add-bot-telegram/

And to allow the bot to post messages you need to make it an admin, follow this guide to do so:<br />
https://www.alphr.com/add-admin-telegram/

#### 4. Add the Telegram channel to the config.yml on your TenderDuty configuration<br />
Last but not least, you need to link the bot and the channel to your TenderDuty configuration. To do so, you need to retrieve the channel ID from your newly created group:<br />
https://stackoverflow.com/questions/45414021/get-telegram-channel-group-id<br />
NOTE: Supergroup and Channel will looks like 1068773197, which is -1001068773197 for bots (with -100 prefix).

You can add the API key and channel ID in the generic part at the start of the config.yml file or set it up per chain.<br />
https://imgur.com/1JHs581
