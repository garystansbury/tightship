# Where TightShip comes from

TightShip generalises an IT-operations suite built and run in production at a county school system
since 2024: a help desk with device-replacement invoicing, Chromebook assignment and fleet tools,
staff account lifecycle, BitLocker key lookup, per-user Wi-Fi keys, DHCP reservations, managed
file transfer, and a self-hosted SAML identity provider for student sign-in with badge, PIN,
security-key and device-certificate factors.

That suite grew tool by tool from two hand-written PHP pages. By its third year it was 154,000
lines, 28 console sections hosted three different ways, six definitions of `isAdmin()`, five
hand-maintained per-role allow-lists, and four separate front doors. It worked, and it was
increasingly hard to reason about.

A first attempt at generalising it (mid-2026) ported eight modules to configuration-driven PHP
with demo and live integration layers, and wrote the capability-layer design that this codebase
adopts. That prototype is archived; its design documents are carried here.

TightShip is the rebuild: a different language, one interface, one permission model, and a shape
that another school system can deploy from a release and a config file. The original suite keeps
running beside it until each module has cut over.
