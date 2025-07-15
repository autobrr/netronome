import { lerpColors } from "tailwind-lerp-colors";
import forms from "@tailwindcss/forms";

const extendedColors = lerpColors();

/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      screens: {
        xs: "400px",
        lg: "1024px",
        xl: "1440px",
        "2xl": "1836px",
        "3xl": "2900px",
      },
      colors: {
        ...extendedColors,
        gray: {
          ...extendedColors.zinc,
          815: "#232427",
        },
      },
      margin: {
        2.5: "0.625rem",
      },
      textShadow: {
        DEFAULT: "0 2px 4px var(--tw-shadow-color)",
      },
      boxShadow: {
        table: "rgba(0, 0, 0, 0.1) 0px 4px 16px 0px",
      },
      animation: {
        fadeIn: "fadeIn 0.5s ease-in-out",
        bounce: "bounce 1s infinite",
        shimmer: "shimmer 2s infinite linear",
        ping: "ping 2s cubic-bezier(0, 0, 0.2, 1) infinite",
      },
      keyframes: {
        fadeIn: {
          "0%": { opacity: "0", transform: "translateY(10px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        bounce: {
          "0%, 100%": {
            transform: "translateY(-25%)",
            animationTimingFunction: "cubic-bezier(0.8, 0, 1, 1)",
          },
          "50%": {
            transform: "translateY(0)",
            animationTimingFunction: "cubic-bezier(0, 0, 0.2, 1)",
          },
        },
        shimmer: {
          "100%": { transform: "translateX(100%)" },
        },
        ping: {
          "75%, 100%": {
            transform: "scale(2)",
            opacity: "0",
          },
        },
      },
    },
  },
  plugins: [forms],
};
