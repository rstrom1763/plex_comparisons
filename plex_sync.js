const fs = require('fs');
const express = require('express');
const app = express();
port = 8081;
app.use(express.json({ limit: '50mb' }));
app.listen(port);

datadir = './data/'
user_data_file = datadir + 'user_data/user_data.json'
console.log('Listening on port ' + port + '... ');

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
    fs.appendFile(user_data_file, JSON.stringify(req.body), 'utf-8', function (err) {
        if (err) throw err;
        res.send('Success');
    });
});