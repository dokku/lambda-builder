require 'json'

def handler(event:, context:)
  logger = Logger.new($stdout)

  logger.info(event)
  logger.info(context)
  "Hello World!"
end