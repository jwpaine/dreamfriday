(function() {
    class ParticleNetworkAnimation {
        constructor() {}

        init(element) {
            this.container = element;
            this.canvas = document.createElement('canvas');
            this.container.appendChild(this.canvas);
            this.ctx = this.canvas.getContext('2d');
            this.resizeCanvas(); // Ensure canvas is sized before creating the network
            this.particleNetwork = new ParticleNetwork(this);

            window.addEventListener('resize', () => this.resizeCanvas());
            return this;
        }

        resizeCanvas() {
            this.canvas.width = this.container.offsetWidth;
            this.canvas.height = this.container.offsetHeight;
            if (this.particleNetwork) {
                this.particleNetwork.recalculateParticles();
            }
        }
    }

    class Particle {
        constructor(parent) {
            this.network = parent;
            this.canvas = parent.canvas;
            this.ctx = parent.ctx;
            this.particleColor = getRandomArrayItem(this.network.options.particleColors);
            this.radius = getLimitedRandom(1.5, 2.5);
            this.angle = Math.random() * Math.PI * 2;
            this.orbitRadius = getLimitedRandom(20, 60);
            this.centerX = Math.random() * this.canvas.width;
            this.centerY = Math.random() * this.canvas.height;
            this.speed = getLimitedRandom(0.005, 0.02);
        }

        update() {
            this.angle += this.speed;
            this.x = this.centerX + Math.cos(this.angle) * this.orbitRadius;
            this.y = this.centerY + Math.sin(this.angle) * this.orbitRadius;
        }

        draw() {
            this.ctx.beginPath();
            this.ctx.fillStyle = this.particleColor;
            this.ctx.globalAlpha = 1;
            this.ctx.arc(this.x, this.y, this.radius, 0, 2 * Math.PI);
            this.ctx.fill();
        }
    }

    class ParticleNetwork {
        constructor(parent) {
            this.options = {
                netLineDistance: 120,
                netLineColor: '#929292',
                particleColors: ['#aaa']
            };
            this.canvas = parent.canvas;
            this.ctx = parent.ctx;
            this.createParticles();
            requestAnimationFrame(() => this.update()); // Ensure animation starts immediately
        }

        createParticles() {
            const quantity = (this.canvas.width * this.canvas.height) / 30000;
            this.particles = Array.from({ length: Math.floor(quantity) }, () => new Particle(this));
        }

        recalculateParticles() {
            this.particles.forEach(p => {
                p.centerX = Math.random() * this.canvas.width;
                p.centerY = Math.random() * this.canvas.height;
            });
        }

        update() {
            this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);

            for (let i = 0; i < this.particles.length; i++) {
                for (let j = i + 1; j < this.particles.length; j++) {
                    let p1 = this.particles[i], p2 = this.particles[j];
                    let dx = p1.x - p2.x, dy = p1.y - p2.y;
                    let distance = Math.sqrt(dx * dx + dy * dy);

                    if (distance < this.options.netLineDistance) {
                        this.ctx.beginPath();
                        this.ctx.strokeStyle = this.options.netLineColor;
                        this.ctx.globalAlpha = (this.options.netLineDistance - distance) / this.options.netLineDistance;
                        this.ctx.lineWidth = 0.7;
                        this.ctx.moveTo(p1.x, p1.y);
                        this.ctx.lineTo(p2.x, p2.y);
                        this.ctx.stroke();
                    }
                }
            }

            this.particles.forEach(p => {
                p.update();
                p.draw();
            });

            requestAnimationFrame(() => this.update());
        }
    }

    function getLimitedRandom(min, max) {
        return Math.random() * (max - min) + min;
    }

    function getRandomArrayItem(array) {
        return array[Math.floor(Math.random() * array.length)];
    }

    document.addEventListener('DOMContentLoaded', () => {
        const element = document.querySelector('.particle-network-animation');
        if (element) {
            new ParticleNetworkAnimation().init(element);
        }
    });
})();
