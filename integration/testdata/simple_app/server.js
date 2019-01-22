const http = require('http')
const port = process.env.PORT || 8080

const requestHandler = (request, response) => {
    return response.end(`STACK: ${process.env.CF_STACK}`)
}

const server = http.createServer(requestHandler);

server.listen(port, (err) => {
    if (err) {
        return console.log('something bad happened', err)
    }
    console.log(`server is listening on ${port}`)
})
