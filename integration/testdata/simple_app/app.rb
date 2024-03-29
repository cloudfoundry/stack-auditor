require 'webrick'

server = WEBrick::HTTPServer.new :Port => ENV['PORT']

server.mount_proc '/' do |request, response|
  response.body = 'Hello World!'
end

trap 'INT' do server.shutdown end

server.start
