import sys
import os
import argparse

sys.path.append(os.path.dirname(__file__))

from service.rgb import RGBService
from follower import LeLampFollower, LeLampFollowerConfig
from lelamp.presets import RGB_CMD_SOLID

def turn_off(port: str, lamp_id: str):
    # Initialize robot connection
    robot_config = LeLampFollowerConfig(port=port, id=lamp_id)
    robot = LeLampFollower(robot_config)
    
    # Initialize RGB service
    rgb_service = RGBService()
    
    try:
        # Connect to robot
        print(f"Connecting to robot on port {port} with ID {lamp_id}...")
        robot.connect(calibrate=False)
        print("Robot connected successfully")
        
        # Start RGB service
        rgb_service.start()
        
        # Turn off LED
        print("Turning off LED")
        rgb_service.dispatch(RGB_CMD_SOLID, (0, 0, 0))
        
        print("Turn off complete")
        
    except Exception as e:
        print(f"Error during turn off: {e}")
    finally:
        # Clean up connections
        if robot.is_connected:
            print("Disconnecting robot...")
            robot.disconnect()
            print("Robot disconnected")
        
        rgb_service.stop()

def main():
    parser = argparse.ArgumentParser(description="Turn off LeLamp LED and disconnect robot")
    parser.add_argument('--id', type=str, required=True, help='ID of the lamp')
    parser.add_argument('--port', type=str, required=True, help='Serial port for the lamp')
    args = parser.parse_args()

    turn_off(args.port, args.id)

if __name__ == "__main__":
    main()
