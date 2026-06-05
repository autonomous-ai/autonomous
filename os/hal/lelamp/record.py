import argparse
import time
import csv
import os
from .leader import LeLampLeader, LeLampLeaderConfig
from lerobot.utils.robot_utils import busy_wait
  
def main():
    parser = argparse.ArgumentParser(description="Check motors status and position")
    parser.add_argument('--id', type=str, required=True, help='ID of the lamp')
    parser.add_argument('--port', type=str, required=True, help='Serial port for the lamp')
    parser.add_argument('--name', type=str, help='Name of recording')
    parser.add_argument('--fps', type=int, default=30, help='Frames per second for recording (default: 30)')
    args = parser.parse_args()

    leader_config = LeLampLeaderConfig(
        port=args.port,
        id=args.id,
    )

    leader = LeLampLeader(leader_config)
    leader.connect(calibrate=False)

    # Wait for user to press enter before starting recording
    input("Press Enter to start recording...")

    # Create recordings directory if it doesn't exist
    recordings_dir = os.path.join(os.path.dirname(__file__), "recordings")
    os.makedirs(recordings_dir, exist_ok=True)

    # Set up CSV file for recording
    csv_filename = os.path.join(recordings_dir, f"{args.name or 'recording'}.csv")
    with open(csv_filename, 'w', newline='') as csvfile:
        csv_writer = None
        
        while True:
            try:
                t0 = time.perf_counter()
                obs = leader.get_action()
                
                # Initialize CSV writer with headers based on first observation
                if csv_writer is None:
                    fieldnames = ['timestamp'] + list(obs.keys()) if isinstance(obs, dict) else ['timestamp', 'observation']
                    csv_writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
                    csv_writer.writeheader()
                
                # Write observation to CSV
                if isinstance(obs, dict):
                    row = {'timestamp': t0, **obs}
                else:
                    row = {'timestamp': t0, 'observation': obs}
                csv_writer.writerow(row)
                csvfile.flush()
                
                print(obs)
                
                # Enforce FPS with busy wait
                busy_wait(1.0 / args.fps - (time.perf_counter() - t0))
                
            except KeyboardInterrupt:
                print("Shutting down teleop...")
                break

if __name__ == "__main__":
    main()