/**
 * Tudou Animation Presets
 *
 * Design direction: Refined, subtle, professional
 * Motion philosophy: Purposeful animations that guide attention, not distract
 */

// Fade variants
export const fade = {
  initial: { opacity: 0 },
  enter: { opacity: 1 },
}

export const fadeUp = {
  initial: { opacity: 0, y: 20 },
  enter: { opacity: 1, y: 0 },
}

export const fadeDown = {
  initial: { opacity: 0, y: -20 },
  enter: { opacity: 1, y: 0 },
}

export const fadeLeft = {
  initial: { opacity: 0, x: -20 },
  enter: { opacity: 1, x: 0 },
}

export const fadeRight = {
  initial: { opacity: 0, x: 20 },
  enter: { opacity: 1, x: 0 },
}

// Scale variants
export const scaleIn = {
  initial: { opacity: 0, scale: 0.95 },
  enter: { opacity: 1, scale: 1 },
}

export const popIn = {
  initial: { opacity: 0, scale: 0.8 },
  enter: { opacity: 1, scale: 1 },
}

// Slide variants
export const slideUp = {
  initial: { opacity: 0, y: 30 },
  enter: {
    opacity: 1,
    y: 0,
    transition: {
      type: 'spring',
      stiffness: 300,
      damping: 30,
    },
  },
}

export const slideInLeft = {
  initial: { opacity: 0, x: -40 },
  enter: {
    opacity: 1,
    x: 0,
    transition: {
      type: 'spring',
      stiffness: 250,
      damping: 25,
    },
  },
}

// Stagger children
export const staggerContainer = {
  initial: {},
  enter: {
    transition: {
      staggerChildren: 0.05,
    },
  },
}

export const staggerItem = {
  initial: { opacity: 0, y: 15 },
  enter: {
    opacity: 1,
    y: 0,
    transition: {
      type: 'spring',
      stiffness: 350,
      damping: 30,
    },
  },
}

// Table row animation
export const tableRow = {
  initial: { opacity: 0, x: -20 },
  enter: {
    opacity: 1,
    x: 0,
    transition: {
      type: 'spring',
      stiffness: 400,
      damping: 35,
    },
  },
}

// Dialog animations
export const dialogOverlay = {
  initial: { opacity: 0 },
  enter: { opacity: 1 },
  leave: { opacity: 0 },
}

export const dialogContent = {
  initial: { opacity: 0, scale: 0.95, y: 20 },
  enter: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: {
      type: 'spring',
      stiffness: 300,
      damping: 30,
    },
  },
  leave: {
    opacity: 0,
    scale: 0.95,
    y: 20,
    transition: {
      duration: 0.15,
    },
  },
}

// Pulse effect for status badges
export const pulse = {
  initial: {},
  enter: {},
  pulsing: {
    scale: [1, 1.05, 1],
    transition: {
      duration: 2,
      repeat: Infinity,
      ease: 'easeInOut',
    },
  },
}

// Button hover
export const buttonHover = {
  initial: { scale: 1 },
  hovered: { scale: 1.02 },
  tapped: { scale: 0.98 },
}

// Success checkmark
export const checkmark = {
  initial: { scale: 0, opacity: 0 },
  enter: {
    scale: 1,
    opacity: 1,
    transition: {
      type: 'spring',
      stiffness: 500,
      damping: 15,
    },
  },
}

// Counter animation for stats
export const counterValue = {
  initial: { opacity: 0, scale: 0.5 },
  enter: {
    opacity: 1,
    scale: 1,
    transition: {
      type: 'spring',
      stiffness: 200,
      damping: 20,
    },
  },
}

// Preset combinations
export const presets = {
  // Page load animation
  pageEnter: {
    initial: { opacity: 0, y: 20 },
    enter: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.4,
        ease: [0.25, 0.46, 0.45, 0.94],
      },
    },
  },

  // List item entrance
  listItem: {
    initial: { opacity: 0, y: 10 },
    enter: {
      opacity: 1,
      y: 0,
      transition: {
        type: 'spring',
        stiffness: 350,
        damping: 30,
      },
    },
  },

  // Card entrance
  cardEnter: {
    initial: { opacity: 0, y: 30, scale: 0.98 },
    enter: {
      opacity: 1,
      y: 0,
      scale: 1,
      transition: {
        type: 'spring',
        stiffness: 200,
        damping: 25,
      },
    },
  },

  // Subtle hover lift
  hoverLift: {
    initial: { y: 0 },
    hovered: {
      y: -2,
      transition: { duration: 0.2 },
    },
  },
}

// Animation timing constants
export const timing = {
  instant: 0,
  fast: 0.15,
  normal: 0.25,
  slow: 0.4,
  slower: 0.6,
}

// Easing functions
export const easing = {
  smooth: [0.25, 0.46, 0.45, 0.94],
  bounce: [0.68, -0.55, 0.27, 1.55],
  spring: { type: 'spring', stiffness: 300, damping: 30 },
  snappy: { type: 'spring', stiffness: 400, damping: 25 },
}
