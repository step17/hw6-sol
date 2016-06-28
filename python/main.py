#!/usr/bin/env python
# -*- coding: utf-8 -*-

import webapp2


class MainPage(webapp2.RequestHandler):
    def get(self):
        self.response.headers['Content-Type'] = 'text/html; charset=utf-8'
        self.response.write("""
        <body>
        <i>Hello world</i> in Japanese is <i>こんにちは世界！</i>
        </body>
        """)
        
app = webapp2.WSGIApplication([
    ('/', MainPage),
], debug=True)
