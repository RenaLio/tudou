/**
 * Tudou Animation Presets
 *
 * Design direction: Refined, subtle, professional
 * Motion philosophy: Purposeful animations that guide attention, not distract
 */

// Fade variants
export const fade = {
  initial: { opacity: 0, },
  enter: { opacity: 1, },
};

export const fadeUp = {
  initial: { opacity: 0, y: 20, },
  enter: { opacity: 1, y: 0, },
};

export const fadeDown = {
  initial: { opacity: 0, y: -20, },
  enter: { opacity: 1, y: 0, },
};

export const fadeLeft = {
  initial: { opacity: 0, x: -20, },
  enter: { opacity: 1, x: 0, },
};

export const fadeRight = {
  initial: { opacity: 0, x: 20, },
  enter: { opacity: 1, x: 0, },
};

// Scale variants
export const scaleIn = {
  initial: { opacity: 0, scale: 0.95, },
  enter: { opacity: 1, scale: 1, },
};

export const popIn = {
  initial: { opacity: 0, scale: 0.8, },
  enter: { opacity: 1, scale: 1, },
};

// Slide variants
export const slideUp = {
  initial: { opacity: 0, y: 30, },
  enter: {
    opacity: 1,
    y: 0,
    transition: {
      type: 'spring',
      stiffness: 300,
      damping: 30,
    },
  },
};

export const slideInLeft = {
  initial: { opacity: 0, x: -40, },
  enter: {
    opacity: 1,
    x: 0,
    transition: {
      type: 'spring',
      stiffness: 250,
      damping: 25,
    },
  },
};

// Stagger children
export const staggerContainer = {
  initial: {},
  enter: {
    transition: {
      staggerChildren: 0.05,
    },
  },
};

export const staggerItem = {
  initial: { opacity: 0, y: 15, },
  enter: {
    opacity: 1,
    y: 0,
    transition: {
      type: 'spring',
      stiffness: 350,
      damping: 30,
    },
  },
};

// Table row animation
export const tableRow = {
  initial: { opacity: 0, x: -20, },
  enter: {
    opacity: 1,
    x: 0,
    transition: {
      type: 'spring',
      stiffness: 400,
      damping: 35,
    },
  },
};

// Dialog animations
export const dialogOverlay = {
  initial: { opacity: 0, },
  enter: { opacity: 1, },
  leave: { opacity: 0, },
};

export const dialogContent = {
  initial: { opacity: 0, scale: 0.95, y: 20, },
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
};

// Pulse effect for status badges
export const pulse = {
  initial: {},
  enter: {},
  pulsing: {
    scale: [1, 1.05, 1,],
    transition: {
      duration: 2,
      repeat: Infinity,
      ease: 'easeInOut',
    },
  },
};

// Button hover
export const buttonHover = {
  initial: { scale: 1, },
  hovered: { scale: 1.02, },
  tapped: { scale: 0.98, },
};

// Success checkmark
export const checkmark = {
  initial: { scale: 0, opacity: 0, },
  enter: {
    scale: 1,
    opacity: 1,
    transition: {
      type: 'spring',
      stiffness: 500,
      damping: 15,
    },
  },
};

// Counter animation for stats
export const counterValue = {
  initial: { opacity: 0, scale: 0.5, },
  enter: {
    opacity: 1,
    scale: 1,
    transition: {
      type: 'spring',
      stiffness: 200,
      damping: 20,
    },
  },
};

// Preset combinations
export const presets = {
  // Page load animation
  pageEnter: {
    initial: { opacity: 0, y: 20, },
    enter: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.4,
        ease: [0.25, 0.46, 0.45, 0.94,],
      },
    },
  },

  // List item entrance
  listItem: {
    initial: { opacity: 0, y: 10, },
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
    initial: { opacity: 0, y: 30, scale: 0.98, },
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
    initial: { y: 0, },
    hovered: {
      y: -2,
      transition: { duration: 0.2, },
    },
  },
};

// Animation timing constants
export const timing = {
  instant: 0,
  fast: 0.15,
  normal: 0.25,
  slow: 0.4,
  slower: 0.6,
};

// Easing functions
export const easing = {
  smooth: [0.25, 0.46, 0.45, 0.94,] as const,
  bounce: [0.68, -0.55, 0.27, 1.55,] as const,
  spring: { type: 'spring' as const, stiffness: 300, damping: 30, },
  snappy: { type: 'spring' as const, stiffness: 400, damping: 25, },
};

// ============================================
// Motion for Vue (motion-v) Presets
// ============================================
// Use with: <motion.div :initial="mvFadeUp.initial" :animate="mvFadeUp.animate" />
// Or: <Motion as="div" v-bind="mvFadeUp" />

export const mvFade = {
  initial: { opacity: 0, },
  animate: { opacity: 1, },
};

export const mvFadeUp = {
  initial: { opacity: 0, y: 20, },
  animate: { opacity: 1, y: 0, },
  transition: { duration: 0.4, ease: easing.smooth, },
};

export const mvFadeDown = {
  initial: { opacity: 0, y: -20, },
  animate: { opacity: 1, y: 0, },
  transition: { duration: 0.4, ease: easing.smooth, },
};

export const mvFadeLeft = {
  initial: { opacity: 0, x: -20, },
  animate: { opacity: 1, x: 0, },
  transition: { duration: 0.4, ease: easing.smooth, },
};

export const mvFadeRight = {
  initial: { opacity: 0, x: 20, },
  animate: { opacity: 1, x: 0, },
  transition: { duration: 0.4, ease: easing.smooth, },
};

export const mvScaleIn = {
  initial: { opacity: 0, scale: 0.95, },
  animate: { opacity: 1, scale: 1, },
  transition: { duration: 0.3, ease: easing.smooth, },
};

export const mvPopIn = {
  initial: { opacity: 0, scale: 0.8, },
  animate: { opacity: 1, scale: 1, },
  transition: { type: 'spring' as const, stiffness: 400, damping: 20, },
};

export const mvSlideUp = {
  initial: { opacity: 0, y: 30, },
  animate: {
    opacity: 1,
    y: 0,
    transition: { type: 'spring' as const, stiffness: 300, damping: 30, },
  },
};

export const mvSlideInLeft = {
  initial: { opacity: 0, x: -40, },
  animate: {
    opacity: 1,
    x: 0,
    transition: { type: 'spring' as const, stiffness: 250, damping: 25, },
  },
};

export const mvStaggerItem = {
  initial: { opacity: 0, y: 15, },
  animate: {
    opacity: 1,
    y: 0,
    transition: { type: 'spring' as const, stiffness: 350, damping: 30, },
  },
};

export const mvTableRow = {
  initial: { opacity: 0, x: -20, },
  animate: {
    opacity: 1,
    x: 0,
    transition: { type: 'spring' as const, stiffness: 400, damping: 35, },
  },
};

// Dialog animations
export const mvDialogOverlay = {
  initial: { opacity: 0, },
  animate: { opacity: 1, },
  exit: { opacity: 0, },
  transition: { duration: 0.2, },
};

export const mvDialogContent = {
  initial: { opacity: 0, scale: 0.95, y: 20, },
  animate: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: { type: 'spring' as const, stiffness: 300, damping: 30, },
  },
  exit: {
    opacity: 0,
    scale: 0.95,
    y: 20,
    transition: { duration: 0.15, },
  },
};

// Hover / press interactions
export const mvHoverLift = {
  whileHover: { y: -2, transition: { duration: 0.2, }, },
  whilePress: { y: 0, scale: 0.98, },
};

export const mvHoverScale = {
  whileHover: { scale: 1.02, transition: { duration: 0.2, }, },
  whilePress: { scale: 0.98, },
};

export const mvHoverGlow = {
  whileHover: {
    boxShadow: '0 0 24px rgba(139, 195, 74, 0.25)',
    transition: { duration: 0.3, },
  },
};

// Scroll-triggered animations
export const mvFadeInView = {
  initial: { opacity: 0, y: 20, },
  whileInView: { opacity: 1, y: 0, },
  viewport: { once: true, margin: '-50px', },
  transition: { duration: 0.5, ease: easing.smooth, },
};

export const mvScaleInView = {
  initial: { opacity: 0, scale: 0.9, },
  whileInView: { opacity: 1, scale: 1, },
  viewport: { once: true, margin: '-50px', },
  transition: { duration: 0.5, ease: easing.smooth, },
};

// Layout animations
export const mvLayout = {
  layout: true,
  transition: { type: 'spring' as const, stiffness: 400, damping: 30, },
};

// Pulse effect
export const mvPulse = {
  animate: {
    scale: [1, 1.05, 1,],
    transition: { duration: 2, repeat: Infinity, ease: 'easeInOut', },
  },
};

// Checkmark pop-in
export const mvCheckmark = {
  initial: { scale: 0, opacity: 0, },
  animate: {
    scale: 1,
    opacity: 1,
    transition: { type: 'spring' as const, stiffness: 500, damping: 15, },
  },
};

// Counter animation
export const mvCounter = {
  initial: { opacity: 0, scale: 0.5, },
  animate: {
    opacity: 1,
    scale: 1,
    transition: { type: 'spring' as const, stiffness: 200, damping: 20, },
  },
};

// Preset combinations for motion-v
export const mvPresets = {
  pageEnter: {
    initial: { opacity: 0, y: 20, },
    animate: {
      opacity: 1,
      y: 0,
      transition: { duration: 0.4, ease: easing.smooth, },
    },
  },

  listItem: {
    initial: { opacity: 0, y: 10, },
    animate: {
      opacity: 1,
      y: 0,
      transition: { type: 'spring' as const, stiffness: 350, damping: 30, },
    },
  },

  cardEnter: {
    initial: { opacity: 0, y: 30, scale: 0.98, },
    animate: {
      opacity: 1,
      y: 0,
      scale: 1,
      transition: { type: 'spring' as const, stiffness: 200, damping: 25, },
    },
  },

  cardHover: {
    ...mvHoverLift,
    ...mvHoverGlow,
  },

  staggerContainer: {
    animate: {
      transition: { staggerChildren: 0.05, },
    },
  },
};

// Re-export motion-v component helpers
export { AnimatePresence, Motion, motion, } from 'motion-v';
