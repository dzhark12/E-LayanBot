import math
import random
import pygame

pygame.init()
W, H = 1100, 850
screen = pygame.display.set_mode((W, H))
pygame.display.set_caption('Celestial Heart | Fireworks Edition')
clock = pygame.time.Clock()
random.seed()
CX, CY, SCALE = W // 2, 420, 175
font = pygame.font.SysFont('consolas', 20)
small = pygame.font.SysFont('consolas', 15)
big = pygame.font.SysFont('georgia', 34, bold=True)
DURATION = 16.0
N = 2600
xs = [-math.sqrt(3) + 2 * math.sqrt(3) * i / (N - 1) for i in range(N)]
bases = [abs(x)**(2/3) for x in xs]
amps = [0.9 * math.sqrt(max(0, 3 - x*x)) for x in xs]
stars = [(random.randrange(W), random.randrange(H), random.uniform(0.6, 2), random.uniform(0, 6.28), random.randrange(90, 220)) for _ in range(240)]
particles = []
shooting = []
rings = []
last_burst = 0.0
finale = False
elapsed = 0.0
paused = False

PALETTE = [(255, 72, 118), (255, 196, 90), (184, 118, 255), (255, 132, 198), (110, 185, 255)]

def add_particle(x, y, vx, vy, color, life, radius=2, gravity=0.0, trail=False):
    particles.append([float(x), float(y), vx, vy, color, life, life, radius, gravity, trail])

def burst(x, y, amount=90, power=180, color=None):
    color = color or random.choice(PALETTE)
    for i in range(amount):
        angle = 2 * math.pi * i / amount + random.uniform(-0.06, 0.06)
        speed = power * random.uniform(0.45, 1.15)
        add_particle(x, y, math.cos(angle)*speed, math.sin(angle)*speed, color,
                     random.uniform(1.1, 2.5), random.choice([1.5, 2, 2.5]), 38, True)
    rings.append([x, y, 0.0, 0.85, color])

def heart_xy(k, idx):
    x = xs[idx]
    y = bases[idx] + amps[idx] * math.sin(k*x)
    return CX + x*SCALE, CY - y*SCALE

def draw_background(t):
    screen.fill((3, 3, 13))
    backdrop = pygame.Surface((W, H), pygame.SRCALPHA)
    pygame.draw.circle(backdrop, (60, 10, 48, 14), (CX, CY), 330)
    pygame.draw.circle(backdrop, (55, 20, 85, 11), (CX, CY), 450)
    screen.blit(backdrop, (0, 0))
    for x, y, radius, phase, brightness in stars:
        twinkle = 0.5 + 0.5*math.sin(t*(0.7+radius*0.3)+phase)
        b = int(brightness*(0.25+0.75*twinkle))
        pygame.draw.circle(screen, (b//2, b//2, b), (int(x), int(y)), max(1, int(radius)))
        if radius > 1.65 and twinkle > 0.82:
            pygame.draw.line(screen, (b//3, b//3, b//2), (x-5,y), (x+5,y))
            pygame.draw.line(screen, (b//3, b//3, b//2), (x,y-5), (x,y+5))

def draw_heart(k, t):
    pulse = 1 + 0.012 * math.sin(t*3.2)
    pts = []
    for x, base, amp in zip(xs, bases, amps):
        y = base + amp*math.sin(k*x)
        pts.append((int(CX + x*SCALE*pulse), int(CY - y*SCALE*pulse)))
    glow = pygame.Surface((W, H), pygame.SRCALPHA)
    pygame.draw.lines(glow, (245, 20, 88, 13), False, pts, 19)
    pygame.draw.lines(glow, (255, 25, 90, 30), False, pts, 9)
    pygame.draw.lines(glow, (255, 75, 130, 75), False, pts, 4)
    screen.blit(glow, (0, 0))
    pygame.draw.aalines(screen, (255, 115, 157), False, pts)
    return pts

def update_particles(dt):
    for p in particles[:]:
        p[0] += p[2]*dt
        p[1] += p[3]*dt
        p[3] += p[8]*dt
        p[2] *= max(0, 1-0.35*dt)
        p[5] -= dt
        if p[5] <= 0:
            particles.remove(p)
    if len(particles) > 2800:
        del particles[:len(particles)-2800]

def draw_particles():
    layer = pygame.Surface((W,H), pygame.SRCALPHA)
    for x,y,vx,vy,color,life,maxlife,radius,gravity,trail in particles:
        if not (-20 < x < W+20 and -20 < y < H+20):
            continue
        a = max(0, min(255, int(255*life/maxlife)))
        r = max(1, int(radius))
        pygame.draw.circle(layer, (*color, min(90,a//3)), (int(x),int(y)), r*3)
        pygame.draw.circle(layer, (*color, a), (int(x),int(y)), r)
        if trail:
            pygame.draw.line(layer, (*color, a//3), (int(x),int(y)),
                             (int(x-vx*.045),int(y-vy*.045)), max(1,r//2))
    screen.blit(layer, (0,0))

def update_rings(dt):
    for r in rings[:]:
        r[2] += 250*dt
        r[3] -= dt
        if r[3] <= 0:
            rings.remove(r)

def draw_rings():
    layer = pygame.Surface((W,H),pygame.SRCALPHA)
    for x,y,radius,life,color in rings:
        pygame.draw.circle(layer, (*color,int(95*life/.85)), (int(x),int(y)), int(radius), 2)
    screen.blit(layer,(0,0))

def spawn_shooting_star():
    shooting.append([random.uniform(0,W*.8), random.uniform(0,H*.35),
                     random.uniform(450,720), random.uniform(180,300), 1.0])

def update_shooting(dt):
    for s in shooting[:]:
        s[0] += s[2]*dt
        s[1] += s[3]*dt
        s[4] -= dt*0.8
        if s[4] <= 0 or s[0] > W+100:
            shooting.remove(s)

def draw_shooting():
    layer = pygame.Surface((W,H),pygame.SRCALPHA)
    for x,y,vx,vy,life in shooting:
        alpha = int(210*life)
        pygame.draw.line(layer,(185,185,255,alpha), (int(x),int(y)),
                         (int(x-vx*.14),int(y-vy*.14)),2)
        pygame.draw.circle(layer,(255,255,255,alpha),(int(x),int(y)),3)
    screen.blit(layer,(0,0))

def draw_ui(k, finale):
    formula = font.render('y = |x|^(2/3) + 0.9 sin(kx) sqrt(3-x^2)', True, (255, 135, 173))
    screen.blit(formula, ((W-formula.get_width())//2, H-107))
    kval = font.render(f'k = {k:06.2f} / 100.00', True, (255, 196, 214))
    screen.blit(kval, ((W-kval.get_width())//2, H-77))
    if finale:
        title = font.render('FOREVER IN THE STARS', True, (255, 215, 155))
        screen.blit(title, ((W-title.get_width())//2, 83))
    hint = small.render('SPACE Pause  |  R Replay  |  F Fireworks  |  ESC Exit', True, (135, 115, 150))
    screen.blit(hint, (20, H-26))

running = True
while running:
    dt = min(clock.tick(60)/1000.0, .05)
    for event in pygame.event.get():
        if event.type == pygame.QUIT:
            running = False
        elif event.type == pygame.KEYDOWN:
            if event.key == pygame.K_ESCAPE:
                running = False
            elif event.key == pygame.K_SPACE:
                paused = not paused
            elif event.key == pygame.K_r:
                elapsed, last_burst, finale = 0.0, 0.0, False
                particles.clear(); rings.clear(); shooting.clear()
                paused = False
            elif event.key == pygame.K_f:
                burst(random.randint(130,W-130),random.randint(110,H-220),130,230)
    if not paused:
        previous = elapsed
        elapsed += dt
        progress = min(elapsed/DURATION,1.0)
        k = 100*(1-(1-progress)**2)
        if random.random() < dt*.32:
            spawn_shooting_star()
        if random.random() < dt*25:
            idx = random.randrange(N)
            hx,hy = heart_xy(k,idx)
            add_particle(hx,hy,random.uniform(-28,28),random.uniform(-55,-12),
                         random.choice([(255,90,130),(255,165,200),(255,220,180)]),
                         random.uniform(.6,1.6),random.choice([1,2]),-8)
        if elapsed > 4 and elapsed-last_burst > (1.1 if finale else 2.5):
            burst(random.randint(100,W-100),random.randint(120,590),
                  130 if finale else 75, 245 if finale else 160)
            last_burst = elapsed
        if not finale and previous < DURATION <= elapsed:
            finale = True
            for x,y in [(CX,175),(CX-300,250),(CX+300,250),(CX-220,480),(CX+220,480)]:
                burst(x,y,170,260)
        update_particles(dt)
        update_rings(dt)
        update_shooting(dt)
    progress = min(elapsed/DURATION,1.0)
    k = 100*(1-(1-progress)**2)
    draw_background(elapsed)
    draw_shooting()
    draw_heart(k,elapsed)
    draw_rings()
    draw_particles()
    draw_ui(k,finale)
    pygame.display.flip()
pygame.quit()
