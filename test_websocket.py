import websockets
import asyncio
import json
import logging

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)

async def test_websocket():
    uri = "ws://localhost:8080/ws?token=abcd"
    async with websockets.connect(uri) as websocket:
        logging.info("Connected to WebSocket server")
        websocket.ping_interval = 30
        websocket.ping_timeout = 10
        welcome_msg = await websocket.recv()
        logging.info(f"Received: {welcome_msg}")

        async def receive_messages():
            while True:
                try:
                    message = await websocket.recv()
                    logging.info(f"Received: {message}")
                except websockets.exceptions.ConnectionClosed:
                    logging.info("Connection closed")
                    break
                except Exception as e:
                    logging.error(f"Error receiving message: {e}")
                    break

        receive_task = asyncio.create_task(receive_messages())

        while True:
            try:
                message = input('("{"id": "your-id", "message": "message"}") or "quit" to exit: ')
                
                if message.lower() == 'quit':
                    break

                try:
                    json.loads(message)
                except json.JSONDecodeError:
                    logging.error("Invalid JSON format. Please try again.")
                    continue

                logging.info(f"Sending message: {message}")
                await websocket.send(message)

            except websockets.exceptions.ConnectionClosed:
                logging.info("Connection closed")
                break
            except Exception as e:
                logging.error(f"Error: {e}")
                break

        receive_task.cancel()
        try:
            await receive_task
        except asyncio.CancelledError:
            pass

async def main():
    try:
        await test_websocket()
    except Exception as e:
        logging.error(f"Main error: {e}")
    finally:
        logging.info("Client shutting down")

if __name__ == "__main__":
    asyncio.run(main())