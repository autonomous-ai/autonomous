from .follower.lelamp_follower import LeLampFollower, LeLampFollowerConfig
import argparse

def main():
    parser = argparse.ArgumentParser(description="Setup motors for LeLamp follower")
    parser.add_argument('--id', type=str, required=True, help='ID of the lamp')
    parser.add_argument('--port', type=str, required=True, help='Serial port for the lamp')
    args = parser.parse_args()

    config = LeLampFollowerConfig(
        port=args.port,
        id=args.id,
    )
    leader = LeLampFollower(config)
    leader.setup_motors()

if __name__ == "__main__":
    main()