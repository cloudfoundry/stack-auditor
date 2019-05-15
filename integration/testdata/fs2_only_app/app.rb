require 'mysql2'
require 'sinatra'

get '/' do
  'Hello World!'
end

get '/version' do
  RUBY_VERSION
end

get '/mysql2' do
  begin
    Mysql2::Client.new(:host => 'testing')
  rescue Mysql2::Error => e
    e.message
  end
end
