/* global google */
/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useState, useRef } from "react";

const AuthContext = createContext();
export function useAuth () {
  return useContext(AuthContext);
}

function loadScript(src) {
  return new Promise((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) return resolve()
    const script = document.createElement('script');
    script.src = src;
    script.onload = () => resolve();
    script.onerror = (err) => reject(err);
    document.body.appendChild(script);
  });
}

export function AuthProvider({config, children}) {
  const [profile, setProfile] = useState();
  const [googleAuthorized, setGoogleAuthorized] = useState();
  const [googleAuthorizationError, setGoogleAuthorizationError] = useState();
  const [googleOAuthState, setGoogleOAuthState] = useState();
  const signInButtonRef = useRef(null);

  function handleAuthenticateResponse(response) {
    setGoogleAuthorized(response.google_authorized);
    setGoogleAuthorizationError(response.google_authorization_error);
    setGoogleOAuthState(response.google_oauth_state);
    setProfile(response.profile);
  }

  const src = 'https://accounts.google.com/gsi/client';
  useEffect(() => {
    if (profile) {
      return;
    }

    // First make a call to the authenticate API in case we have
    // previously authenticated, and have a cookie.
    fetch("/api/authenticate").then(response => {
      if (response.ok) {
        response.json().then(data => handleAuthenticateResponse(data));
      } else {
        loadScript(src).then(() => {
          google.accounts.id.initialize({
            client_id: config.google.client_id,
            auto_select: true,
            context: "signin",
            use_fedcm_for_prompt: true,
            callback: (response) => {
              const params = {
                headers: {Authorization: "Bearer " + response.credential}
              };
              return fetch("/api/authenticate", params).then(response => {
                if (response.ok) {
                  response.json().then(data => handleAuthenticateResponse(data));
                }
              });
            },
          });
          google.accounts.id.renderButton(signInButtonRef.current, {});
          google.accounts.id.prompt((notification) => {
            // Just log for debugging, the header button will handle manual sign-in
            if (notification.isNotDisplayed()) {
              const reason = notification.getNotDisplayedReason();
              console.log('Google Sign-In prompt not displayed:', reason);
            }
          });
        }).catch(console.error)
      }
    });

    return () => {
      const scriptTag = document.querySelector(`script[src="${src}"]`)
      if (scriptTag) document.body.removeChild(scriptTag)
    }
  }, [profile, config.google.client_id, signInButtonRef]);

  function authorizeGoogle() {
    const scope = encodeURI(config.google.oauth_scope);
    const url = "https://accounts.google.com/o/oauth2/v2/auth?response_type=code&access_type=offline&prompt=consent&" +
                `redirect_uri=${window.location.origin}/api/oauth/google&` +
                `client_id=${config.google.client_id}&` +
                `login_hint=${profile.email}&` +
                `scope=${scope}&` +
                `state=${googleOAuthState}`;
    window.location.href = url;
  }

  const value = {
    profile,
    authorizeGoogle,
    googleAuthorized, googleAuthorizationError, googleOAuthState,
    signInButtonRef,
  };
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
