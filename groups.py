import random
import time

pool1 = ["Asgard Ymir", "Overwork", "Plate Group", "Los Pinguinos"]
pool2 = ["Alpha Impact", "UW Paradox", "TST", "munich eSport Celestial"]
pool3 = ["ushi's Daycare", "MRG Amethyst", "Sovereign", "Warrior Genesis"]
pool4 = ["UKN Lupine", ".eXe", "Kiira", "Wave Racers"]

pools = [pool1, pool2, pool3, pool4]
g1, g2, g3, g4 = [], [], [], []

groups = [g1, g2, g3, g4]
group_amount = 4
group_size = 4

for i in range(group_amount):
    for j in range(group_size):
        element = random.choice(pools[j])
        groups[i].append(element)
        pools[j].remove(element)

for i, group in enumerate(groups):
    print(f"Group {i+1}: {group}")
    time.sleep(5)