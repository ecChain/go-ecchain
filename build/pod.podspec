Pod::Spec.new do |spec|
  spec.name         = 'Gec'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/ecchain/go-ecchain'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS ecchain Client'
  spec.source       = { :git => 'https://github.com/ecchain/go-ecchain.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gec.framework'

	spec.prepare_command = <<-CMD
    curl https://gecstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gec.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
