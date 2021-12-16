const fs = require('fs');
const http = require('http');
const https = require('https');
const express = require('express');
const privateKey = fs.readFileSync('server.key', 'utf8');
const certificate = fs.readFileSync('server.crt', 'utf8');
const credentials = { key: privateKey, cert: certificate };
const app = express();
app.use(express.json({ limit: '50mb' }));
port = 8081;

datadir = './data/'
user_data_dir = datadir + 'user_data/'

app.get('/test', (req, res) => {
    res.send("Success")
});

app.post('/sync', (req, res) => {

    fs.writeFile(datadir + req.headers.token + '.json', JSON.stringify(req.body), (err) => err);
    res.send("Sync Successful");

});

app.get('/diff', (req, res) => {

    fs.readFile(datadir + req.headers.token + '.json', 'utf8', (err, data) => {

        if (err) {
            res.send(err);
            console.log(err)
            return;
        }
        else {
            res.send(JSON.stringify(data));
        };

    });

});

app.post('/newuser', (req, res) => {
    //route for creating a new user
    //uses token header created by client to name file
    fs.writeFile(user_data_dir + req.headers.access_token + '.json', JSON.stringify(req.body), 'utf-8', (err) => {

        if (err) throw err;
        res.send('Success');

    });

});

var httpServer = http.createServer(app);
var httpsServer = https.createServer(credentials, app);
httpsServer.listen(port);
console.log('Listening on port ' + port + '... ');
