'use client'

import Image from 'next/image'
import { useState } from 'react'
import heroImg from '../src/assets/hero.png'

function ReactMark({ className, label }: { className: string; label?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 256 228"
      role={label ? 'img' : 'presentation'}
      aria-label={label}
      aria-hidden={label ? undefined : true}
    >
      <ellipse cx="128" cy="114" rx="95" ry="36" fill="none" stroke="#00d8ff" strokeWidth="12" />
      <ellipse
        cx="128"
        cy="114"
        rx="95"
        ry="36"
        fill="none"
        stroke="#00d8ff"
        strokeWidth="12"
        transform="rotate(60 128 114)"
      />
      <ellipse
        cx="128"
        cy="114"
        rx="95"
        ry="36"
        fill="none"
        stroke="#00d8ff"
        strokeWidth="12"
        transform="rotate(120 128 114)"
      />
      <circle cx="128" cy="114" r="14" fill="#00d8ff" />
    </svg>
  )
}

export default function HomePage() {
  const [count, setCount] = useState(0)

  return (
    <main className="site-shell">
      <section id="center">
        <div className="hero">
          <Image src={heroImg} className="base" width={170} height={179} alt="" priority />
          <ReactMark className="framework" label="React logo" />
          <span className="next-logo" aria-hidden="true">N</span>
        </div>
        <div>
          <h1>Get started</h1>
          <p>
            Edit <code>app/page.tsx</code> and save to test <code>Fast Refresh</code>
          </p>
        </div>
        <button
          type="button"
          className="counter"
          onClick={() => setCount((currentCount) => currentCount + 1)}
        >
          Count is {count}
        </button>
      </section>

      <div className="ticks" />

      <section id="next-steps">
        <div id="docs">
          <svg className="icon" role="presentation" aria-hidden="true">
            <use href="/icons.svg#documentation-icon" />
          </svg>
          <h2>Documentation</h2>
          <p>Your questions, answered</p>
          <ul>
            <li>
              <a href="https://nextjs.org/docs" target="_blank" rel="noreferrer">
                <span className="button-icon next-mark" aria-hidden="true">N</span>
                Explore Next.js
              </a>
            </li>
            <li>
              <a href="https://react.dev/" target="_blank" rel="noreferrer">
                <ReactMark className="button-icon" />
                Learn React
              </a>
            </li>
          </ul>
        </div>
        <div id="social">
          <svg className="icon" role="presentation" aria-hidden="true">
            <use href="/icons.svg#social-icon" />
          </svg>
          <h2>Connect with us</h2>
          <p>Join the React community</p>
          <ul>
            <li>
              <a href="https://github.com/vercel/next.js" target="_blank" rel="noreferrer">
                <svg className="button-icon" role="presentation" aria-hidden="true">
                  <use href="/icons.svg#github-icon" />
                </svg>
                GitHub
              </a>
            </li>
            <li>
              <a href="https://nextjs.org/discord" target="_blank" rel="noreferrer">
                <svg className="button-icon" role="presentation" aria-hidden="true">
                  <use href="/icons.svg#discord-icon" />
                </svg>
                Discord
              </a>
            </li>
            <li>
              <a href="https://x.com/nextjs" target="_blank" rel="noreferrer">
                <svg className="button-icon" role="presentation" aria-hidden="true">
                  <use href="/icons.svg#x-icon" />
                </svg>
                X.com
              </a>
            </li>
          </ul>
        </div>
      </section>

      <div className="ticks" />
      <section id="spacer" />
    </main>
  )
}
