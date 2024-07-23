const { match } = require('assert');
const { EmbedBuilder, Client, GatewayIntentBits, time } = require('discord.js');
const Discord = require("discord.js");
const client = new Client({ intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildMessages, GatewayIntentBits.MessageContent] });
const fs = require('fs');
const path = require('path');

const token = 'TOKEN';
const logChannelID = '1255266356455018506'
let logChannel;
let embedColor = "78DFEA";
const pugsChannelID = '1265099445935280149';
let pugsChannel;
let pugsPlayers = [];
const ALLOWED_USERS = ['429302329188286495', '646251699706527745', '732178580729102359'];

client.once('ready', () => {
    console.log(`Logged in as ${client.user.tag}!`);

    logChannel = client.channels.cache.get(logChannelID);
    if (!logChannel) {
        console.error('Log channel not found');
    }
});

const scheduledRemovals = {};

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
        .setColor(embedColor)
        .setDescription(`${message.content}`)
        .setAuthor({
            name: message.author.username,
            iconURL: message.author.displayAvatarURL({ dynamic: true }),
        });
        logChannel.send({embeds: [embed]})
            .catch(console.error); 
        }

    else if (message.content === "!schedule") {
        const embed = new EmbedBuilder().setTitle('Saltwater Showdown Schedule')
        .setColor('#7289da')
        .setDescription('Group Stage: July 15th\n\nPlayoffs: August 5th\nGrandfinals: To be decided')
        message.channel.send({embeds: [embed]});
    }   

    if (message.content === "!pugs help" || message.content == "!pugs") {
        const embed = new EmbedBuilder().setTitle('Saltwater Showdown PUGs Bot Guide')
        .setColor(embedColor)
        .setDescription('!pugs join -> Join Pugs\n\n!pugs quit -> Quit Pugs\n\n!pugs list -> List all currently signed up players\n\n!pugs rules -> Get the format and rules of our PUGs\n\nYou are removed from the PUGs list an hour after signing up.\n\nEveryone on the list will be pinged once 10 players sign up.');
        message.channel.send({embeds: [embed]});
    }   

    else if (message.content === "!pugs rules") {
        const embed = new EmbedBuilder().setTitle('**How to PUG**')
        .setColor(embedColor)
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
        .setColor(embedColor)
        .setDescription(str);
        message.channel.send({embeds: [embed]});

    }

    else if (message.content.startsWith("!pstats")) {

        let parts = message.content.split(' ');

        if (!parts[1]) {
            message.channel.send('```!pstats usage: <Player Name>```');
            return;
        }
        
        const response = await fetch(`http://localhost:8080/getPlayerStats?player=${parts[1]}`);
    
        // Check if response is OK
        if (!response.ok) {
          throw new Error(`Network response was not ok: ${response.statusText}`);
        }
        
        // Parse JSON response
        const data = await response.json();
        
        // Create embed and send message
        const embed = new EmbedBuilder()
          .setTitle(parts[1])
          .setColor(embedColor)
          .setDescription(data.message || 'No data message found');
        
        await message.channel.send({ embeds: [embed] });
        
    }

   
        // Check if the message starts with !uploadMap
    else if (message.content.startsWith('!uploadMap')) {
          // Ensure the user is one of the allowed users
          if (!ALLOWED_USERS.includes(message.author.id)) {
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
                const response = await fetch(`http://localhost:8080/createMap?matchID=${matchID}&winner=${winner}&map=${mapName}&fileName=${fileName}`);
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
            if (!ALLOWED_USERS.includes(message.author.id)) {
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

        else if (message.content === '!commands' || message.content === '!help') {

            const bendixID = "429302329188286495";
            const bendix = await client.users.fetch(bendixID);
        
                const embed = new EmbedBuilder().setTitle('The Lagoon Bot Commands')
                .setAuthor({
                    name: bendix.username,
                    iconURL: bendix.displayAvatarURL({ dynamic: true }),
                })
                .setColor(embedColor)
                .setDescription('Commands:\n\n'
                + '!help / !commands: Lists all available bot commands\n'
                + '!pugs help: Lists all available pugs commands\n'
                + '!pstats <Player OW Name>: Returns player stats\n\n'
                + '!rules / !rulebook\n'
                + '!dates / !schedule\n'
                + '!standings\n'
                + '!report\n'
                + '!appeal\n'
                + '!socials\n\n'
                + 'If any questions persist, please feel free to contact our server staff.');
                message.channel.send({embeds: [embed]});
            }

        else if (message.content === '!rules' || message.content === '!rulebook') {
            message.channel.send("Click here to view our rules:\nhttps://docs.google.com/document/d/1AYav_lBA2OHSC5Djm9pmyNDqC_rusuHbLUtcmIDFqv8/edit?usp=sharing");
        }
    
        else if (message.content === '!dates' || message.content === '!schedule') {
            message.channel.send('Group Stages start July 15th.\n\nPlayoffs start August 5th. Grandfinals date is still up for decision.')
        }
    
        else if (message.content === '!standings') {
            message.channel.send("https://docs.google.com/spreadsheets/d/1Xnfz2-KbzwwhOC4E47WvBvK7CxaX9Tq_XAHcOwSfjBU/edit?usp=sharing");
        }
    
        else if (message.content === '!signup') {
            message.channel.send("Click here to sign up your team:\nhttps://forms.gle/zQFQKAK1ozn9XEZo6");
        }
    
        else if (message.content === '!report') {
            message.channel.send("Click here to submit a report:\nhttps://forms.gle/B14dDynVwfiRCvEm8");
        }
    
        else if (message.content === '!appeal') {
            message.channel.send("Click here to appeal a staff decision:\nhttps://forms.gle/nPx5UzrRyixuVuQj8")
        }
    
        else if (message.content === '!staff') {
            message.channel.send("Insert list of all staff");
        }
    
        else if (message.content === '!socials') {
            message.channel.send("Insert links to all socials");
        }
    
        else if (message.content === '!twitch') {
            message.channel.send("https://twitch.tv/saltwatershowdown");
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
    
        else if (message.content === '!podcast' || message.content === '!news' || message.content === '!saltynewsnetwork' || message.content === '!snn') {
            message.channel.send("We run a weekly news show, the Salty News Network, to keep our members updated on all the news surrounding the tournament! Ask our staff for more info and let us know if you want to get involved!");
        }
    
        else if (message.content === '!lagoon' || message.content === '!thelagoon' || message.content === '!org') {
            message.channel.send("This tournament is hosted by The Lagoon! Insert discord link");
        }

        else if (message.content.startsWith('!')) {
            const bendixID = "429302329188286495";
            const bendix = await client.users.fetch(bendixID);
        
                const embed = new EmbedBuilder()
                .setAuthor({
                    name: bendix.username,
                    iconURL: bendix.displayAvatarURL({ dynamic: true }),
                })
                .setColor(embedColor)
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