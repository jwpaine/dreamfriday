(function() {
    class ParticleNetworkAnimation {
        constructor() {}

        init(element) {
            this.container = element;
            this.canvas = document.createElement('canvas');
            this.sizeCanvas();
            this.container.appendChild(this.canvas);
            this.ctx = this.canvas.getContext('2d');
            this.particleNetwork = new ParticleNetwork(this);

            this.bindUiActions();
            return this;
        }

        bindUiActions() {
            window.addEventListener('resize', () => {
                this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
                this.sizeCanvas();
                this.particleNetwork.createParticles();
            });
        }

        sizeCanvas() {
            this.canvas.width = this.container.offsetWidth;
            this.canvas.height = this.container.offsetHeight;
        }
    }

    class Particle {
        constructor(parent, x, y) {
            this.network = parent;
            this.canvas = parent.canvas;
            this.ctx = parent.ctx;
            this.particleColor = getRandomArrayItem(this.network.options.particleColors);
            this.radius = getLimitedRandom(1.5, 2.5);
            this.opacity = 0;
            this.x = x || Math.random() * this.canvas.width;
            this.y = y || Math.random() * this.canvas.height;
            this.velocity = {
                x: (Math.random() - 0.5) * parent.options.velocity,
                y: (Math.random() - 0.5) * parent.options.velocity
            };
        }

        update() {
            this.opacity = Math.min(this.opacity + 0.01, 1);
            if (this.x > this.canvas.width + 100 || this.x < -100) this.velocity.x *= -1;
            if (this.y > this.canvas.height + 100 || this.y < -100) this.velocity.y *= -1;
            this.x += this.velocity.x;
            this.y += this.velocity.y;
        }

        draw() {
            this.ctx.beginPath();
            this.ctx.fillStyle = this.particleColor;
            this.ctx.globalAlpha = this.opacity;
            this.ctx.arc(this.x, this.y, this.radius, 0, 2 * Math.PI);
            this.ctx.fill();
        }
    }

    class ParticleNetwork {
        constructor(parent) {
            this.options = {
                velocity: 1,
                density: 30000,
                netLineDistance: 200,
                netLineColor: '#929292',
                particleColors: ['#aaa']
            };
            this.canvas = parent.canvas;
            this.ctx = parent.ctx;
            this.init();
        }

        init() {
            this.createParticles(true);
            this.animationFrame = requestAnimationFrame(this.update.bind(this));
            this.bindUiActions();
        }

        createParticles(isInitial) {
            this.particles = [];
            const quantity = (this.canvas.width * this.canvas.height) / this.options.density;
            if (isInitial) {
                let counter = 0;
                clearInterval(this.createIntervalId);
                this.createIntervalId = setInterval(() => {
                    if (counter < quantity - 1) {
                        this.particles.push(new Particle(this));
                    } else {
                        clearInterval(this.createIntervalId);
                    }
                    counter++;
                }, 250);
            } else {
                for (let i = 0; i < quantity; i++) {
                    this.particles.push(new Particle(this));
                }
            }
        }

        update() {
            this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
            this.ctx.globalAlpha = 1;

            for (let i = 0; i < this.particles.length; i++) {
                for (let j = this.particles.length - 1; j > i; j--) {
                    let p1 = this.particles[i], p2 = this.particles[j];
                    let distance = Math.sqrt((p1.x - p2.x) ** 2 + (p1.y - p2.y) ** 2);
                    if (distance > this.options.netLineDistance) continue;

                    this.ctx.beginPath();
                    this.ctx.strokeStyle = this.options.netLineColor;
                    this.ctx.globalAlpha = ((this.options.netLineDistance - distance) / this.options.netLineDistance) * p1.opacity * p2.opacity;
                    this.ctx.lineWidth = 0.7;
                    this.ctx.moveTo(p1.x, p1.y);
                    this.ctx.lineTo(p2.x, p2.y);
                    this.ctx.stroke();
                }
            }

            this.particles.forEach(p => {
                p.update();
                p.draw();
            });

            if (this.options.velocity !== 0) {
                this.animationFrame = requestAnimationFrame(this.update.bind(this));
            } else {
                cancelAnimationFrame(this.animationFrame);
            }
        }

        bindUiActions() {
            this.canvas.addEventListener('mousemove', (e) => {
                if (!this.interactionParticle) {
                    this.createInteractionParticle();
                }
                this.interactionParticle.x = e.offsetX;
                this.interactionParticle.y = e.offsetY;
            });

            this.canvas.addEventListener('mouseout', () => {
                this.removeInteractionParticle();
            });
        }

        createInteractionParticle() {
            this.interactionParticle = new Particle(this);
            this.interactionParticle.velocity = { x: 0, y: 0 };
            this.particles.push(this.interactionParticle);
            return this.interactionParticle;
        }

        removeInteractionParticle() {
            let index = this.particles.indexOf(this.interactionParticle);
            if (index > -1) {
                this.particles.splice(index, 1);
                this.interactionParticle = undefined;
            }
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
