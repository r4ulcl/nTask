import csv
import time
import random
import sys

def generate_random_data():
    return [random.randint(1, 100), random.uniform(0.0, 1.0), random.choice(['A', 'B', 'C'])]

def main():

    print(sys.argv[1:])

    header = ['h1', 'h2', 'h3']

    # Generate random data
    data = generate_random_data()

    # Write to CSV
    print(header)
    print(data)

    # Sleep for a random time between 1 and 30 seconds
    sleep_time = random.uniform(5, 15)
    print(f"Sleeping for {sleep_time:.2f} seconds...")
    time.sleep(sleep_time)

if __name__ == "__main__":
    main()
