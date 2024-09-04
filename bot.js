const { EmbedBuilder, Client, GatewayIntentBits } = require('discord.js');
const fsp = require('fs').promises;
const fs = require('fs')
const path = require('path');


const client = new Client({ intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildMessages, GatewayIntentBits.MessageContent] });

const configFile = "config.json";
const token = "TOKEN";

let logChannelID, pugsChannelID;
let logChannel, pugsChannel;
let pugsPlayers = [];
let scheduledRemovals = [];

async function readConfig() {
    try {
        const jsonString = await fsp.readFile(configFile, 'utf8');
        return JSON.parse(jsonString);
    } catch (err) {
        console.error('Error reading or parsing file', err);
        throw err; // Ensure errors are propagated
    }
}

async function writeConfig(data) {
    try {
        await fsp.writeFile(configFile, JSON.stringify(data, null, 2), 'utf8');
        console.log('File updated successfully');
    } catch (err) {
        console.error('Error writing file', err);
    }
}

async function addAllowedUser(newUser) {
    const data = await readConfig();
    if (!data.allowedUsers.includes(newUser)) {
        data.allowedUsers.push(newUser);
        await writeConfig(data);
    }
}

async function isUserAllowed(user) {
    const data = await readConfig();
    return data.allowedUsers.includes(user);
}

async function setLogChannel(channelID) {
    const data = await readConfig();
    data.logChannelID = channelID;
    await writeConfig(data);
}

async function getLogChannel() {
    const data = await readConfig();
    return data.logChannelID;
}

async function setPugsChannel(channelID) {
    const data = await readConfig();
    data.pugsChannelID = channelID;
    await writeConfig(data);
}

async function getPugsChannel() {
    const data = await readConfig();
    return data.pugsChannelID;
}

async function setEmbedColor(colorCode) {
    const data = await readConfig();
    data.embedColor = colorCode;
    await writeConfig(data);
}

async function getEmbedColor() {
    const data = await readConfig();
    return data.embedColor;
}

client.once('ready', async () => {
    console.log(`Logged in as ${client.user.tag}!`);

    logChannelID = await getLogChannel();
    pugsChannelID = await getPugsChannel();

    logChannel = client.channels.cache.get(logChannelID);
    if (!logChannel) {
        console.error('Log channel not found');
    }

    pugsChannel = client.channels.cache.get(pugsChannelID);
    if (!pugsChannel) {
        console.error('PUGs channel not found');
    }
});

client.login(token).catch(err => console.error('Failed to login', err));


function removePlayerFromPugs(players, player) {
    const index = players.indexOf(player);
    if (index !== -1) {
        players.splice(index, 1);
        pugsChannel.send(`Removed ${player} from the PUGs list!`);
    }
}

function scheduleRemoval(arr, value, delay) {
    const timeoutId = setTimeout(() => {
        removePlayerFromPugs(arr, value);
        delete scheduledRemovals[value.id];
    }, delay);

    scheduledRemovals[value.id] = timeoutId;
}

function cancelScheduledRemoval(value) {
    const timeoutId = scheduledRemovals[value.id];
    if (timeoutId) {
        clearTimeout(timeoutId);
        delete scheduledRemovals[value.id];
    }
}

function cancelAllScheduledRemovals() {
    for (let playerId in scheduledRemovals) {
        if (scheduledRemovals.hasOwnProperty(playerId)) {
            clearTimeout(scheduledRemovals[playerId]);
        }
    }
    Object.keys(scheduledRemovals).forEach(playerId => delete scheduledRemovals[playerId]);
}

client.on('messageCreate', async message => {
  
    if (message.author.bot) return;

    if (logChannel) {
        const embed = new EmbedBuilder().setTitle(`<#${message.channelId}>`)
        .setColor(await getEmbedColor())
        .setDescription(`${message.content}`)
        .setAuthor({
            name: message.author.username,
            iconURL: message.author.displayAvatarURL({ dynamic: true }),
        });
        logChannel.send({embeds: [embed]})
            .catch(console.error); 
        }
    
    if (message.content.startsWith("!setLogChannel")) {

        if (!await isUserAllowed(message.author.id)) {
            message.channel.send("You don't have permission to perform this action");
            return;
        }
        await setLogChannel(message.channel.id);
    }

    else if (message.content.startsWith("!setPugsChannel")) {

        if (!await isUserAllowed(message.author.id)) {
            message.channel.send("You don't have permission to perform this action");
            return;
        }
        await setPugsChannel(message.channel.id);
    }

    else if (message.content.startsWith("!setEmbedColor")) {

        let parts = message.content.split(' ');

        if (!await isUserAllowed(message.author.id)) {
            message.channel.send("You don't have permission to perform this action");
            return;
        }
        await setEmbedColor(parts[1]);
        const embed = new EmbedBuilder().setTitle('**Title**')
        .setColor(await getEmbedColor())
        .setDescription('You changed the embed color! Good job pookiebear');
        message.channel.send({embeds: [embed]});
    }

    else if (message.content.startsWith("!addAdmin")) {
        let parts = message.content.split(' ');
        if (!await isUserAllowed(message.author.id)) {
            message.channel.send("You don't have permission to perform this action");
            return;
        }
        await addAllowedUser(parts[1].replace("<", "").replace(">", "").replace("@", ""));
        message.channel.send("Admin has been added");
    }

    else if (message.content === "!pugs help" || message.content == "!pugs") {
        const embed = new EmbedBuilder().setTitle('Saltwater Showdown PUGs Bot Guide')
        .setColor(await getEmbedColor())
        .setDescription('!pugs join -> Join Pugs\n\n!pugs quit -> Quit Pugs\n\n!pugs list -> List all currently signed up players\n\n!pugs rules -> Get the format and rules of our PUGs\n\nYou are removed from the PUGs list an hour after signing up.\n\nEveryone on the list will be pinged once 10 players sign up.');
        message.channel.send({embeds: [embed]});
    }   

    else if (message.content === "!pugs rules") {
        const embed = new EmbedBuilder().setTitle('**How to PUG**')
        .setColor(await getEmbedColor())
        .setDescription('- 2 captains have to be picked\n- The captains join the pick room\n\n- The captains decide on sides\n\n- Captain A picks 1 player\n- Captain B picks 1 player\n- Captain A picks 2 players\n- Captain B picks 2 players\n- Captain A picks 2 players\n- Captain B picks 2 players\n- Captain A picks the gamemode\n- Captain B picks a map');
        message.channel.send({embeds: [embed]});
    }

    else if (message.content === "!pugs join") {

        if (pugsPlayers.indexOf(message.author) !== -1) {
            message.channel.send("You are already signed up for PUGs!");
            return;
        }

        pugsPlayers.push(message.author);
        message.channel.send('You are now signed up for PUGs!');
        scheduleRemoval(pugsPlayers, message.author, 1000 * 60 * 60);

        if (pugsPlayers.length == 10) {
            
            let str = "**PUGs are starting!**\n\n";

            if (pugsPlayers.length == 0) {
                message.channel.send("No players are currently signed up for PUGs.");
                return;
            }

            for (let index = 0; index < pugsPlayers.length; index++) {
                const element = `${pugsPlayers[index]}`;
                str += element + "\n";
                str.replace("undefined", "");
            }

            pugsChannel.send(str);

            cancelAllScheduledRemovals();
            pugsPlayers = [];
        }
    }

    else if (message.content === "!pugs quit" || message.content === "!pugs leave") {
        let index = pugsPlayers.indexOf(message.author);
        if (index !== -1) {
            pugsPlayers.splice(index, 1);
            message.channel.send("You have been removed from the PUGs list!");
            cancelScheduledRemoval(message.author);
        } else {
            message.channel.send("You are not signed up for PUGs!");
        }
    }

    else if (message.content === "!pugs list") {

        let str = "";

        if (pugsPlayers.length == 0) {
            message.channel.send("No players are currently signed up for PUGs.");
            return;
        }

        for (let index = 0; index < pugsPlayers.length; index++) {
            const element = `${pugsPlayers[index].username}`;
            str += element + "\n";
            str.replace("undefined", "");
        }

        const embed = new EmbedBuilder().setTitle('PUGs Player List')
        .setColor(await getEmbedColor())
        .setDescription(str);
        message.channel.send({embeds: [embed]});

    }

    else if (message.content.startsWith("!pstats")) {

        let parts = message.content.split(' ');

        if (!parts[1]) {
            message.channel.send('```!pstats usage: <Player Name> [Hero Name]```');
            return;
        }

        if (parts[2]) {

            const response = await fetch(`http://localhost:8080/hStats?player=${parts[1]}&hero=${parts[2]}`);
            const data = await response.json();

            const embed = new EmbedBuilder()
            .setTitle(parts[1] + " on " + parts[2])
            .setColor(await getEmbedColor())
            .setDescription(data.message || 'No data message found');
            
            await message.channel.send({ embeds: [embed] });
            return;

        }

        const response = await fetch(`http://localhost:8080/pStats?player=${parts[1]}`);
        const data = await response.json();

        const embed = new EmbedBuilder()
        .setTitle(parts[1])
        .setColor(await getEmbedColor())
        .setDescription(data.message || 'No data message found');
        
        await message.channel.send({ embeds: [embed] });
        return;
    
    }

    else if (message.content.startsWith("!tstats")) {

        let parts = message.content.split(' ');

        if (!parts[1]) {
            message.channel.send('```!tstats usage: <Team Name> [Map Name] -- Replace spaces with "_"```');
            return;
        }

        if (parts[2]) {

            const response = await fetch(`http://localhost:8080/tmStats?team=${parts[1]}&map=${parts[2]}`);
            const data = await response.json();

            const embed = new EmbedBuilder()
            .setTitle(parts[1].replace('_', ' ') + " on " + parts[2].replace('_', ' '))
            .setColor(await getEmbedColor())
            .setDescription(data.message || 'No data message found');
            
            await message.channel.send({ embeds: [embed] });
            return;

        }

        const response = await fetch(`http://localhost:8080/tStats?team=${parts[1]}`);
        const data = await response.json();

        const embed = new EmbedBuilder()
        .setTitle(parts[1].replace('_', ' '))
        .setColor(await getEmbedColor())
        .setDescription(data.message || 'No data message found');
        
        await message.channel.send({ embeds: [embed] });
        return;
    
    }

    else if (message.content.startsWith('!compareStats')) {
        let parts = message.content.split(' ');
        if (parts.length !== 3) {
            message.channel.send('Usage: !compareStats <Player 1> <Player 2>');
            return;
        }

        const response = await fetch(`http://localhost:8080/compareStats?player1=${parts[1]}&player2=${parts[2]}`);

        const data = await response.json();
        message.channel.send(`\`\`\`${data.message}\`\`\``);
    }

        // Check if the message starts with !uploadMap
    else if (message.content.startsWith('!uploadMap')) {
          // Ensure the user is one of the allowed users
          if (!await isUserAllowed(message.author.id)) {
            return message.channel.send('You are not authorized to use this command.');
          }
      
          // Split the command arguments
          let args = message.content.split(' ');
          if (args.length !== 4) {
            return message.channel.send('Usage: !uploadMap <matchID> <mapName> <winner>');
          }
      
          let [command, matchID, mapName, winner] = args;
      
          // Check if there are attachments
          if (message.attachments.size === 0) {
            return message.channel.send('Please attach a text file.');
          }
      
          // Process each attachment
          message.attachments.forEach(async (attachment) => {
            if (attachment.name.endsWith('.txt')) {
              try {
                let fileName = attachment.name;
                const filePath = path.join(__dirname, fileName);
                fileName = fileName.replace(".txt", "");
      
                // Download and save the file
                await downloadFile(attachment.url, filePath);
      
                // Make the fetch request
                const response = await fetch(`http://localhost:8080/uploadMap?matchID=${matchID}&winner=${winner}&map=${mapName}&fileName=${fileName}`);
                if (!response.ok) {
                  message.channel.send('Internal Server Error');
                  return;
                }
      
                const data = await response.json();
                message.channel.send(`${data.message}`);
              } catch (error) {
                console.error('Error:', error);
                message.channel.send('An error occurred while processing the file.');
              }
            } else {
              message.channel.send('Only text files are allowed.');
            }
          });
        }

        else if (message.content.startsWith('!createMatch')) {
            // Ensure the user is one of the allowed users
            if (!await isUserAllowed(message.author.id)) {
              return message.channel.send('You are not authorized to use this command.');
            }
        
            // Split the command arguments
            let args = message.content.split(' ');
        
            let [command, team1, team2, grandfinals] = args;
        
            try {
              // Make the fetch request
              const response = await fetch(`http://localhost:8080/createMatch?team1=${team1}&team2=${team2}&grandfinals=${grandfinals}`);
              if (!response.ok) {
                throw new Error(`Network response was not ok: ${response.statusText}`);
              }
        
              const data = await response.json();
              message.channel.send(`${data.message}`);
            } catch (error) {
              console.error('Error:', error);
              message.channel.send('An error occurred while creating the match.');
            }
        }

        else if (message.content.startsWith("!updateLeaderboards")) {
            if (!await isUserAllowed(message.author.id)) {
                return message.channel.send('You are not authorized to use this command.');
              }
            
              try {
                // Make the fetch request
                const response = await fetch(`http://localhost:8080/updateLeaderboards`);
                if (!response.ok) {
                  throw new Error(`Network response was not ok: ${response.statusText}`);
                }
          
                const data = await response.json();
                message.channel.send(`${data.message}`);
              } catch (error) {
                console.error('Error:', error);
                message.channel.send('An error occured updating the leaderboards.');
              }
        }
        
        else if (message.content === '!commands' || message.content === '!help') {

            const bendixID = "429302329188286495";
            const bendix = await client.users.fetch(bendixID);
        
                const embed = new EmbedBuilder().setTitle('Bot Commands')
                .setAuthor({
                    name: bendix.username,
                    iconURL: bendix.displayAvatarURL({ dynamic: true }),
                })
                .setColor(await getEmbedColor())
                .setDescription('Commands:\n\n'
                + '!help / !commands: Lists all available bot commands\n'
                + '!pugs help: Lists all available pugs commands\n'
                + '!comparestats <Player 1 OW Name> <Player 2 OW Name>: Compares two players\n'
                + '!tstats <Team> (optional: <Map>): Returns team stats -- Spaces replaced by underscore\n'
                + '!pstats <Player OW Name> (optional: <Hero>): Returns player stats -- Spaces replaced by underscore\n\n'
                + '!rules / !rulebook\n'
                + '!dates / !schedule\n'
                + '!standings\n'
                + '!report\n'
                + '!appeal\n'
                + '!socials\n\n'
                + 'If any questions persist, please feel free to contact our server staff.');
                message.channel.send({embeds: [embed]});
            }

        else if (message.content === '!admin') {

            const bendixID = "429302329188286495";
            const bendix = await client.users.fetch(bendixID);
        
                const embed = new EmbedBuilder().setTitle('Admin Commands')
                .setAuthor({
                    name: bendix.username,
                    iconURL: bendix.displayAvatarURL({ dynamic: true }),
                })
                .setColor(await getEmbedColor())
                .setDescription('Commands:\n\n'
                + '!setLogChannel\n'
                + '!setPugsChannel\n'
                + '!setEmbedColor <Hexcode without #>\n'
                + '!updateLeaderboards\n'
                + '!addAdmin\n\n'
                + '!createMatch [Team1] [Team2] [0 / 1 if GF] -> spits out matchID REPLACE SPACE WITH UNDERSCORE\n'
                + '!uploadMap [matchID] [Map] [Winner] REPLACE SPACE WITH UNDERSCORE'
                )
                message.channel.send({embeds: [embed]});
        }

        else if (message.content === '!rules' || message.content === '!rulebook') {
            message.channel.send("Insert rulebook");
        }
    
        else if (message.content === '!dates' || message.content === '!schedule') {
            message.channel.send('Insert schedule')
        }
    
        else if (message.content === '!standings') {
            message.channel.send("Insert standings link");
        }
    
        else if (message.content === '!signup') {
            message.channel.send("Insert signup form");
        }
    
        else if (message.content === '!report') {
            message.channel.send("Insert report form");
        }
    
        else if (message.content === '!appeal') {
            message.channel.send("Insert appeal form")
        }
    
        else if (message.content === '!staff') {
            message.channel.send("Insert list of all staff");
        }
    
        else if (message.content === '!socials') {
            message.channel.send("Insert links to all socials");
        }
    
        else if (message.content === '!twitch') {
            message.channel.send("Insert twitch link");
        }
    
        else if (message.content === '!twitter') {
            message.channel.send("Insert twitter link");
        }
    
        else if (message.content === '!tiktok') {
            message.channel.send("Insert tiktok link");
        }
    
        else if (message.content === '!youtube') {
            message.channel.send("Insert youtube link");
        }
    
        else if (message.content === '!clip') {
            message.channel.send("Insert clip submission form");
        }
    
        else if (message.content === '!goat') {
            message.channel.send("```Bendix.```");
        }

        else if (message.content.startsWith('!')) {
            const bendixID = "429302329188286495";
            const bendix = await client.users.fetch(bendixID);
        
                const embed = new EmbedBuilder()
                .setAuthor({
                    name: bendix.username,
                    iconURL: bendix.displayAvatarURL({ dynamic: true }),
                })
                .setColor(await getEmbedColor())
                .setDescription("Use !help or !commands for a list of commands.");
                message.channel.send({embeds: [embed]});
        }
      });
        




function downloadFile(url, filePath) {
    return new Promise((resolve, reject) => {
      const mod = url.startsWith('https') ? require('https') : require('http');
      mod.get(url, (response) => {
        if (response.statusCode === 200) {
          const file = fs.createWriteStream(filePath);
          response.pipe(file);
          file.on('finish', () => {
            file.close(() => resolve());
          });
        } else {
          reject(new Error(`Failed to download file. Status code: ${response.statusCode}`));
        }
      }).on('error', (err) => {
        reject(new Error(`Error downloading file: ${err.message}`));
      });
    });
}

client.login(token);